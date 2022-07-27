module.exports = async ({ github, context, npmDiff, npmChalk, diffable, outputType, outputFormat }) => {
  const d = JSON.parse(diffable)

  const diff = makeDiff(npmDiff, d)

  const markdownDiffWithHeader = '### Flux Kustomization diffs\n\n' + diff[1]

  if (outputType === 'pr_comment' && diff[1] !== '') {
    github.rest.issues.createComment({
      issue_number: context.issue.number,
      owner: context.repo.owner,
      repo: context.repo.repo,
      body: markdownDiffWithHeader
    })
  }

  if (outputFormat === 'markdown') {
    if (diff[1] === '') {
      return ''
    }

    return markdownDiffWithHeader
  }

  return colorizeDiff(diff[0], npmChalk)
}

function makeDiff (npmDiff, diffable) {
  let rawDiff = ''
  let markdownDiff = ''

  for (const mapping of diffable.mappings) {
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

  return [rawDiff, markdownDiff]
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
