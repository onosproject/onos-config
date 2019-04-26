``` 
go get https://github.com/openconfig/gnmi
go get github.com/openconfig/ygot/ygot

git clone https://github.com/openconfig/gnmi.git
cd gnmi/testing/fake/gnmi/cmd/fake_server/
```

fake server:
https://devcenter.heroku.com/articles/ssl-certificate-self
Run first steps form here
https://github.com/openconfig/gnmi/tree/master/testing/fake/gnmi/cmd/fake_server
```
--- a/testing/fake/gnmi/cmd/gen_fake_config/gen_config.go
+++ b/testing/fake/gnmi/cmd/gen_fake_config/gen_config.go
@@ -34,7 +34,7 @@ var (
        outputPath = "config.pb.txt"

        config = &fpb.Config{
-               Target: "fake target name",
+               Target: "Test-onos-config",
```

```
./fake_server --config config.pb.txt --text --port 8080 --server_crt server.crt --server_key server.key --allow_no_client_auth --logtostderr
```

then 

```
cd onos-config/southbound
go run *.go
```