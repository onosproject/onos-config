# Issues and Pull Requests

The onos-config project uses GitHub Issues to track work items and bugs that were discovered. 
Instructions on this page show how to relate Issues to Pull Requests during the 
development workflow.

## Issues 
Issues are units of work to be done in the onos project and can represent new features to be developed
or defects to be fixed.

### Find an Issue
You can browse [through the existing issues](https://github.com/onosproject/onos-config/issues) 
or you can _search_ for a specific one.

### Open an Issue
If you want to work on a new feature that is not yet tracked, please create a new issue to represent
the work and assign it to an appropriate project, e.g. Core, Northbound.

### Work on an Issue
After you found or created an issue to work on, you should:
* assign that issue to yourself
* go to the [projects](https://github.com/onosproject/onos-config/projects)
* select the project you have assigned the issue to
* drag and drop the issue to the `In Progress` column to let people know that you are working on it 

## Pull Requests
This section describes how to open a pull request and assign it to one of the several projects in
onos-config.

### Reference an Issue from Your Commit
If you have an issue identifying your work in onos-config [issues](https://github.com/onosproject/onos-config/issues), 
To automatically link your pull request an issue, before pushing a commit to your fork of 
onos-config please insert `fixes #<issue-number>` into the commit message.

The following is an example of a complete commit message:
```
Adding pull request workflow
Fixes #90

# Please enter the commit message for your changes. Lines starting
# with '#' will be ignored, and an empty message aborts the commit.
#
# Date:      Fri May 10 11:36:26 2019 +0200
#
# On branch pr-process
# Changes to be committed:
#       new file:   docs/Pull_requests.md
#
# Changes not staged for commit:
#       modified:   docs/Pull_requests.md
#
```

### Open a Pull Request
When you open a pull request for _myfeature_ you need to add the PR to a `project` (e.g. Northbound) through the github UI. 
Please also assign a reviewer out of the suggested ones. If none are suggested please pick one from the core team.     

More information on opening pull requests can be found [in the GitHub documentation](https://help.github.com/en/articles/creating-a-pull-request).


### Track a Pull Request
After your pull request is included into a onos-config _project_ you can find it under the `In Progress` tab.  
At this point in time the PR will go through a lifecycle:
* Review from different people --> your PR will go into `Review in progress` state
* if changes are requested you will have to go back and address them
* when your PR is approved and Testing is passed it will go under `Reviewer approved` state
* after the PR is in `Reviewer approved` it can be merged
* when the PR is merged both the PR and the issue will move under `Done` state