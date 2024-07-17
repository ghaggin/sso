# SSO

Implementation of a test Identity Provider and Service Provider

## Setup
Generate self-signed key pair for the service provider
```lang=bash
./gen_certs.sh sp localhost:8123
```

Generate self-signed key pair for the identity provider
```lang=bash
./gen_certs.sh idp localhost:8124
```

## Test SP<>IDP integration
1. Run IDP
```
go run main.go -mode idp
```
2. Run SP
```
go run main.go -mode sp
```
Warning: IDP login will not work until the SP is added as a service provider to the IDP

3. Upload SP metadata.xml file to IDP by running both the SP and the IDP and then running these commands
```
curl -s http://localhost:8123/saml/metadata > tmp/sp_metadata.xml
curl localhost:8124/service -d @tmp/sp_metadata.xml
```
4. Navigate to [http://localhost:8123](http://localhost:8123) to test the login flow
