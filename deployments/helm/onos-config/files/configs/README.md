This folder contains test configurations that are loaded into the application
via a `ConfigMap` when deploying via this Helm chart. The root test configurations
are copied here because Helm does not allow files outside the chart to be accessed
via templates. Changes to the root test configurations should be reflected here.

Do not copy the **deviceStore-sample.json** file in here - it will be generated
automatically from [configmap.yaml](../../templates/configmap.yaml)
