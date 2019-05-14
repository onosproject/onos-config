# Developer Workflow

> Steps outlined in this page assume that the [development prerequisites](prerequisites.md) are met. 

## Create Workspace
Before making your first contribution, you should follow the steps to 
[create your own GitHub fork](contributing.md#1-fork-on-github) and 
[local workspace as outlined in the contributor guide](contributing.md#2-clone-fork).

After this, you can browse the code and start making changes as necessary.

## Build and Test
After you made some changes to the code locally, before opening a pull request you should run 
a few steps to make sure the code will pass validation by the CI:

* Run and pass `make build`
* Run and pass `make test`

You can find more information on the full build process in the [building onos-config](build.md) document.

## Submit a Pull Request
If the build and the test passed, you can commit your code and open a new pull request 
as described in more detail in the [contributing to onos-config](contributing.md#5-commit) document.

## Pull Request Review process
The pull request you just opened will be checked by our Travis CI system and reviewed by the community. 
Once it is approved, it will be merged it with a _squash and merge_ strategy. 
If you are requested for changes in your pull request please go back and start again with [step number 4 
in the contributing to onos-config guide](contributing.md#4-keep-branch-in-sync).


