# Developer Workflow for onos-config

Note: This file assumes you have passed all the steps in [Prerequisites](prerequisites.md) and checked out the code as described in the first 4 steps in [the contributing how-to](contributing.md).

## Docker
See [the docker build README](/../master/build/dev-docker/README.md) for instructions
to build a Dev image that downloads any dependencies to you local folder
This resolves any go dependencies that are needed. 

## Building and Running Tests

After you made some changes to the code locally before opening a pull request you should run a few steps:

* Run and pass `make build`
* Run and pass `make test`

You can find more information in the [build file](build.md)

## Submit a Pull Request

After the build and the test pass you can open a pull request following steps from 5 onwards of the [contributing how-to](contributing.md).

## Pull Request Review process
The pull Request you just opened will be checked by our Travis CI system and reviewed by the community. 
Once it it's approved it will be merged it with a _squash and merge_ strategy. 
If you are requested for changes in your pull request please go back and start again for step number 4 in the [contributing how-to](contributing.md).

