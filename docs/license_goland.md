# GoLand Copyright Profile Setup

GoLand IDE can be configured to automatically include the required Apache 2.0 license text in
Go source files. Steps to do this are shown below:

* Open `Preferences` window
* Select `Editor->Copyright->Copyright Profiles`
* Click the `plus` icon
* Name the new profie `onos`
* Cut and paste the text from [here](./license_goland.txt) into the copyright field
* Click `Apply`
* Open `Formatting` then `Go`
* Click check box for `Use Custom Formatting Options`
* Click checkbox `Use Line Comment`
* Click `Apply`

Once the new copyright profile is created, it needs to be applied to the project:
* Select `Editor->Copyright`
* Choose `onos` in the `Default Project Copyright` drop-down list
* Click `OK`