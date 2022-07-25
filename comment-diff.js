module.exports = async ({ github, context, npmDiff, npmChalk, diffable, outputType, outputFormat }) => {
  const d = JSON.parse(diffable)
  let rawDiff = ''
  let markdownDiff = '### Flux Kustomization diffs\n\n'

  // TODO: don't output empty diffs
  for (const mapping of d.mappings) {
    const diff = npmDiff.createTwoFilesPatch(
      mapping.srcPath,
      mapping.dstPath,
      mapping.srcContent,
      mapping.dstContent
    )
    rawDiff += diff
    markdownDiff += '<details>\n  <summary>diff: ' +
                    mapping.SrcPath + ' | ' +
                    mapping.DstPath +
                    '</summary>\n\n```diff\n' +
                    diff +
                    '```\n</details>\n'
  }

  if (outputType === 'pr_comment') {
    github.rest.issues.createComment({
      issue_number: context.issue.number,
      owner: context.repo.owner,
      repo: context.repo.repo,
      body: markdownDiff
    })
  }

  if (outputFormat === 'markdown') {
    return markdownDiff
  }

  const colorizedDiff = colorizeDiff(rawDiff, npmChalk)

  return colorizedDiff
}

function colorizeDiff (rawDiffStr, chalk) {
  const chalkBasicColor = new chalk.Instance({ level: 1 })
  const rawDiffArr = rawDiffStr.split(/\r?\n/)

  let colorizedDiffStr = ''
  for (const line of rawDiffArr) {
    if (line.startsWith('+')) {
      colorizedDiffStr += chalkBasicColor.green(line) + '\n'
    } else if (line.startsWith('-')) {
      colorizedDiffStr += chalkBasicColor.red(line) + '\n'
    } else {
      colorizedDiffStr += line + '\n'
    }
  }

  return colorizedDiffStr
}
