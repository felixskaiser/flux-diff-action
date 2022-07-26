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
	"strings"

	"github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/fluxcd/kustomize-controller/api/v1beta2"
	flag "github.com/spf13/pflag"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

var (
	srcDir string
	dstDir string
)

// go run get-diffable.go --src=dummy-test-repo/clusters/production --dst=dummy-test-repo/clusters/staging
func main() {
	flag.StringVar(&srcDir, "src", "", "Path to the source directory.")
	flag.StringVar(&dstDir, "dst", "", "Path to the destination directory.")
	flag.Parse()

	if srcDir == "" {
		log.Fatal("flag 'src' must not be empty")
	}

	if dstDir == "" {
		log.Fatal("flag 'dst' must not be empty")
	}

	d, err := findBuildAll(srcDir, dstDir)
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
	fullName   string
	kustPath   string
	sourcePath string
}

func findBuildAll(srcDir, dstDir string) (DiffableList, error) {
	var d DiffableList

	c, err := compareDirs(srcDir, dstDir)
	if err != nil {
		return d, fmt.Errorf("error finding and building Flux Kustomizations: %v", err)
	}

	for _, item := range c.items {
		var srcKustPath, dstKustPath string

		if item.src.kustPath != "" {
			srcKustPath = item.src.kustPath
		}

		if item.dst.kustPath != "" {
			dstKustPath = item.dst.kustPath
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

func compareDirs(srcDir, dstDir string) (comparisonList, error) {
	srcKusts, err := findInDir(srcDir)
	if err != nil {
		return comparisonList{}, fmt.Errorf("error comparing directories: %v", err)
	}

	dstKusts, err := findInDir(dstDir)
	if err != nil {
		return comparisonList{}, fmt.Errorf("error comparing directories: %v", err)
	}

	srcMap, dstMap, err := makeMapRec(srcKusts, dstKusts, 0)
	if err != nil {
		return comparisonList{}, fmt.Errorf("error comparing directories: %v", err)
	}

	return compareMaps(srcMap, dstMap), nil
}

func findInDir(dirPath string) ([]fluxKust, error) {
	fk := []fluxKust{}

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

		//TODO: also handle target namespace?
		var v1beta2K v1beta2.Kustomization
		err = yaml.UnmarshalStrict(fileContent, &v1beta2K)
		if err == nil && v1beta2K.TypeMeta.Kind == "Kustomization" && v1beta2K.TypeMeta.APIVersion == "kustomize.toolkit.fluxcd.io/v1beta2" {
			fk = append(fk, fluxKust{
				fullName:   fmt.Sprintf("%s/%s", v1beta2K.ObjectMeta.Namespace, v1beta2K.ObjectMeta.Name),
				kustPath:   v1beta2K.Spec.Path, //TODO: handle 'None'
				sourcePath: p,
			})
			return nil
		}

		//TODO: also handle target namespace?
		var v1beta1K v1beta1.Kustomization
		err = yaml.UnmarshalStrict(fileContent, &v1beta1K)
		if err == nil && v1beta1K.TypeMeta.Kind == "Kustomization" && v1beta1K.TypeMeta.APIVersion == "kustomize.toolkit.fluxcd.io/v1beta1" {
			fk = append(fk, fluxKust{
				fullName:   fmt.Sprintf("%s/%s", v1beta1K.ObjectMeta.Namespace, v1beta1K.ObjectMeta.Name),
				kustPath:   v1beta1K.Spec.Path, //TODO: handle 'None'
				sourcePath: p,
			})
			return nil
		}

		return nil
	})

	return fk, err
}

func makeMapRec(srcKusts, dstKusts []fluxKust, depth int) (map[string]fluxKust, map[string]fluxKust, error) {
	srcMap := make(map[string]fluxKust)
	dstMap := make(map[string]fluxKust)

	for _, kust := range srcKusts {
		key, err := makeUniqueKey(kust, depth)
		if err != nil {
			return srcMap, dstMap, fmt.Errorf("cannot make map with unique keys: %v", err)
		}

		if _, ok := srcMap[key]; ok {
			depth++
			return makeMapRec(srcKusts, dstKusts, depth)
		}
		srcMap[key] = kust
	}

	for _, kust := range dstKusts {
		key, err := makeUniqueKey(kust, depth)
		if err != nil {
			return srcMap, dstMap, fmt.Errorf("cannot make map with unique keys: %v", err)
		}

		if _, ok := dstMap[key]; ok {
			depth++
			return makeMapRec(srcKusts, dstKusts, depth)
		}
		dstMap[key] = kust
	}

	return srcMap, dstMap, nil
}

func makeUniqueKey(kust fluxKust, depth int) (string, error) {
	splitPath := strings.Split(kust.sourcePath, "/")
	if (depth + 1) >= len(splitPath) {
		return "", fmt.Errorf("kustomization %s at %s is not unique", kust.fullName, kust.sourcePath)
	}

	pathParts := splitPath[len(splitPath)-(depth+1) : len(splitPath)-1]
	keyPrefix := strings.Join(pathParts, "_")

	//TODO: also use kustPath

	return fmt.Sprintf("%s_%s", keyPrefix, kust.fullName), nil
}

func compareMaps(srcMap, dstMap map[string]fluxKust) comparisonList {
	var c comparisonList

	// compare src against dst
	for k, srcV := range srcMap {
		if dstV, ok := dstMap[k]; ok {
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
	for k, dstV := range dstMap {
		if _, ok := srcMap[k]; !ok {
			c.items = append(c.items, comparison{
				name: k,
				src:  fluxKust{},
				dst:  dstV,
			})
		}
	}

	return c
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
