module.exports = async ({ github, context, npmDiff, npmChalk, diffable, outputType, outputFormat }) => {
  const d = JSON.parse(diffable)

  let rawDiff = ''
  let markdownDiff = ''

  for (const mapping of d.mappings) {
    const diff = npmDiff.createTwoFilesPatch(
      mapping.srcPath,
      mapping.dstPath,
      mapping.srcContent,
      mapping.dstContent
    )

    const emptyDiff = '===================================================================\n' + 
                      '--- ' + mapping.srcPath + '\n' +
                      '+++ ' + mapping.dstPath + '\n'

    if (diff !== emptyDiff) {
      rawDiff += diff
      markdownDiff += '<details>\n  <summary>diff: ' +
                      mapping.srcPath + ' | ' +
                      mapping.dstPath +
                      '</summary>\n\n```diff\n' +
                      diff +
                      '```\n</details>\n'
    }
  }

  const markdownDiffWithHeader = '### Flux Kustomization diffs\n\n' + markdownDiff

  if (outputType === 'pr_comment' && markdownDiff !== '') {
    github.rest.issues.createComment({
      issue_number: context.issue.number,
      owner: context.repo.owner,
      repo: context.repo.repo,
      body: markdownDiffWithHeader
    })
  }

  if (outputFormat === 'markdown') {
    if (markdownDiff === '') {
      return ''
    }

    return markdownDiffWithHeader
  }

  const colorizedDiff = colorizeDiff(rawDiff, npmChalk)

  return colorizedDiff
}

function colorizeDiff (rawDiffStr, chalk) {
  if (rawDiffStr === '') {
    return ''
  }

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
