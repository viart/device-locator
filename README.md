# Apple Devices Locator

```
echo -n | openssl s_client -servername fmipmobile.icloud.com -host fmipmobile.icloud.com -port 443 -prexit -showcerts 2>/dev/null | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > fmipmobile.crt
```
