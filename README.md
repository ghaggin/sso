# Test IDP/SP

## Setup
Generate self-signed key pair for the service provider
```lang=bash
./gen_certs.sh sp localhost:8123
```

Generate self-signed key pair for the service provider
```lang=bash
./gen_certs.sh idp localhost:8124
```

## Run SP with samltest.id
Upload metadata to samltest.id.  
1. Run task1 to generate metadata file
2. Navigate to https://samltest.id/upload.php
3. Upload file sp_metadata.xml

## Run local SP
```
go run main.go --mode sp --port 8123 --idp http://localhost:8124/metadata
```
Note: idp can be the metadata endpoint for samltest.id as well

## Run local IDP
```
go run main.go --mode idp --port 8124
```

Upload SP metadata.xml file to IDP by running both the SP and the IDP and then running these commands
```
curl http://localhost:8123/saml/metadata > sp_metadata.xml
curl localhost:8124/service -d @sp_metadata.xml
```
