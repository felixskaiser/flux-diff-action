/*
Copyright 2022 Felix Kaiser

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/fluxcd/kustomize-controller/api/v1beta2"
	flag "github.com/spf13/pflag"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

var (
	srcDir  string
	dstDir  string
	workDir string
)

// go run get-diffable.go --src=dummy-test-repo/clusters/production --dst=dummy-test-repo/clusters/staging
func main() {
	flag.StringVar(&srcDir, "src", "", "Path to the source directory.")
	flag.StringVar(&dstDir, "dst", "", "Path to the destination directory.")
	flag.StringVar(&workDir, "workDir", "", "Path to the working directory.")
	flag.Parse()

	if srcDir == "" {
		log.Fatal("flag 'src' must not be empty")
	}

	if dstDir == "" {
		log.Fatal("flag 'dst' must not be empty")
	}

	d, err := FindBuildAll(srcDir, dstDir, workDir)
	if err != nil {
		log.Fatalf("error finding and building all Flux Kustomizations: %s", err)
	}

	out, err := json.Marshal(d)
	if err != nil {
		log.Fatalf("error marshalling JSON: %s", err)
	}

	fmt.Println(string(out))
}

type DiffableList struct {
	Mappings []Diffable `json:"mappings"`
}

type Diffable struct {
	SrcPath    string `json:"srcPath"`
	DstPath    string `json:"dstPath"`
	SrcContent string `json:"srcContent"`
	DstContent string `json:"dstContent"`
}

type comparisonList struct {
	items []comparison
}

type comparison struct {
	name string
	src  fluxKust
	dst  fluxKust
}

type fluxKust struct {
	kustPath   string
	sourcePath string
}

func FindBuildAll(srcDir, dstDir, workDir string) (DiffableList, error) {
	var d DiffableList

	c, err := compare(srcDir, dstDir)
	if err != nil {
		return d, fmt.Errorf("error finding and building Flux Kustomizations: %v", err)
	}

	for _, item := range c.items {
		srcKustPath := workDir
		dstKustPath := workDir

		if item.src.kustPath != "" {
			srcKustPath = filepath.Join(srcKustPath, item.src.kustPath)
			log.Printf("DEBUG: found srcKustPath: %s", srcKustPath)
		}

		if item.dst.kustPath != "" {
			dstKustPath = filepath.Join(dstKustPath, item.dst.kustPath)
			log.Printf("DEBUG: found dstKustPath: %s", dstKustPath)
		}

		srcYaml, err := kustomizeBuild(srcKustPath)
		if err != nil {
			return d, fmt.Errorf("error finding and building Flux Kustomizations: %v", err)
		}

		dstYaml, err := kustomizeBuild(dstKustPath)
		if err != nil {
			return d, fmt.Errorf("error finding and building Flux Kustomizations: %v", err)
		}

		d.Mappings = append(d.Mappings, Diffable{
			SrcPath:    item.src.sourcePath,
			DstPath:    item.dst.sourcePath,
			SrcContent: srcYaml,
			DstContent: dstYaml,
		})
	}

	return d, nil
}

func compare(srcDir, dstDir string) (comparisonList, error) {
	//TODO: also handle target namespace?
	var c comparisonList

	srcKusts, err := findInDir(srcDir)
	if err != nil {
		return c, fmt.Errorf("error comparing directories: %v", err)
	}

	dstKusts, err := findInDir(dstDir)
	if err != nil {
		return c, fmt.Errorf("error comparing directories: %v", err)
	}

	// TODO: break if namespace/name combinations are not uniqe per dir/cluster
	// compare src against dst
	for k, srcV := range srcKusts {
		if dstV, ok := dstKusts[k]; ok {
			c.items = append(c.items, comparison{
				name: k,
				src:  srcV,
				dst:  dstV,
			})
		} else {
			c.items = append(c.items, comparison{
				name: k,
				src:  srcV,
				dst:  fluxKust{},
			})
		}
	}

	// compare remainder of dst against src
	for k, dstV := range dstKusts {
		if _, ok := srcKusts[k]; !ok {
			c.items = append(c.items, comparison{
				name: k,
				src:  fluxKust{},
				dst:  dstV,
			})
		}
	}

	return c, nil
}

func findInDir(dirPath string) (map[string]fluxKust, error) {
	fMap := make(map[string]fluxKust)

	err := filepath.WalkDir(dirPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error finding Flux Kustomizations in directory: %v", err)
		}

		re := regexp.MustCompile(`\.ya?ml`)
		hasYamlFileExt := re.MatchString(d.Name())

		if d.IsDir() || !(hasYamlFileExt) {
			return nil
		}

		fileContent, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("error reading file: %v", err)
		}

		var v1beta2K v1beta2.Kustomization
		err = yaml.Unmarshal(fileContent, &v1beta2K)
		if err == nil {
			fMap[fmt.Sprintf("%s/%s", v1beta2K.Namespace, v1beta2K.Name)] = fluxKust{
				kustPath:   v1beta2K.Spec.Path, //TODO: handle 'None'
				sourcePath: p,
			}

			return nil
		}

		var v1beta1K v1beta1.Kustomization
		err = yaml.Unmarshal(fileContent, &v1beta1K)
		if err == nil {
			fMap[fmt.Sprintf("%s/%s", v1beta1K.Namespace, v1beta1K.Name)] = fluxKust{
				kustPath:   v1beta1K.Spec.Path, //TODO: handle 'None'
				sourcePath: p,
			}

			return nil
		}

		return nil
	})

	return fMap, err
}

func kustomizeBuild(dirPath string) (string, error) {
	if dirPath == "" {
		return "", nil
	}

	kustomizer := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	fs := filesys.MakeFsOnDisk()

	resMap, err := kustomizer.Run(fs, dirPath)
	if err != nil {
		return "", fmt.Errorf("error performing kustomization: %v", err)
	}

	resYaml, err := resMap.AsYaml()
	if err != nil {
		return "", fmt.Errorf("error returning resource as yaml: %v", err)
	}

	return string(resYaml), nil
}
