# Gitattributes

The `.gen.go` files from the previous exercises now live in your repository, and generated files in version control need a bit of care. The [`.gitattributes`](https://git-scm.com/docs/gitattributes) file we added tells Git and GitHub how to treat them. It helps with two problems: teammates trying to resolve merge conflicts in generated code, and PR reviews cluttered with hundreds of lines of [oapi-codegen](https://academy.threedots.tech/knowledge/openapi) output.

## Merge Conflicts in Generated Code

Consider two developers working on separate branches. Both modify the OpenAPI spec and run `go generate ./...`.
Both branches now have a different version of the same `openapi.gen.go` file, fully rewritten by oapi-codegen.
When they merge, Git shows dozens of conflicting lines in the generated files.

**The conflict looks scary, but nobody should resolve it by hand.** **Instead of resolving the conflicts, regenerate the file.**
Merge the OpenAPI specs, then run `go generate ./...`, and commit the output. That's the correct workflow for any generated file.

Note that `.gitattributes` doesn't prevent this merge conflict. It still happens.
The `linguist-generated` attribute is not a merge strategy.
What it does is communicate to your team that this file is machine output, so nobody wastes time reviewing or hand-merging it line by line.

## PR Review Noise

What's worse, generated files clutter every pull request.
Your change touches 15 lines in the OpenAPI spec and 20 lines in the handler, but the diff also includes 223 lines of regenerated boilerplate.
Reviewers have to scroll past generated files trying to find your actual code.

Your project's `.gitattributes` file contains a single line:

```text
**/**.gen.go linguist-generated=true
```

This glob pattern matches any `.gen.go` file at any directory depth. `linguist-generated=true` tells [GitHub Linguist](https://github.com/github-linguist/linguist) (and GitLab) that these files are generated. GitHub **collapses them by default** in the "Files changed" tab and excludes them from language statistics. Reviewers see only the files that matter and can still expand the generated file if they need to.

## Why Commit Generated Files?

The alternative is adding `.gen.go` to `.gitignore` and regenerating in CI.
This complicates the developer experience, since you need to always run `go generate` before building the project.
The CI jobs take longer.
You also lose the ability to track how the generated code changes over time.

**For most projects, committing generated files and marking them with `.gitattributes` is the best approach.**

## Asserting no changes

We recommend running the code generators in the CI and asserting that nothing changed.
For example, you can check that `git status --porcelain` is empty.
This way, you can catch if someone forgets to run `go generate` after changing the OpenAPI spec.

Note you need to use the same exact version of the tools in CI as everyone do locally for this to work.
This is now trivial with `go tool` support in `go.mod`.

{{tip}}

Next time you open a PR on GitHub with generated Go files, notice how they're collapsed by default. For more on keeping PRs reviewable, see our [How to Create PRs That Get Merged The Same Day](https://threedots.tech/episode/prs-that-get-merged-the-same-day/) episode.

{{endtip}}

## Exercise

Exercise path: ./project

We've added a `.gitattributes` file to your project that marks all `.gen.go` files as generated.
Nothing else to do for now.
