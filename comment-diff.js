module.exports = async ({ github, context, npmDiff, npmChalk, diffable, outputType, outputFormat }) => {
  const d = JSON.parse(diffable)
  let rawDiff = ''
  let markdownDiff = '### Flux Kustomization diffs\n\n'

  for (const mapping of d.Mappings) {
    const diff = npmDiff.createTwoFilesPatch(
      mapping.SrcPath,
      mapping.DstPath,
      mapping.SrcContent,
      mapping.DstContent
    )
    rawDiff += diff
    markdownDiff += '<details>\n  <summary>diff: ' +
                    mapping.SrcPath + ' | ' +
                    mapping.DstPath +
                    '</summary>\n\n```diff\n' +
                    rawDiff +
                    '```\n</details>'
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
