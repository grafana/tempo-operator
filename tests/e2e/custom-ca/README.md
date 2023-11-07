# Example certs
The example certs were generated with the following commands:

openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes -keyout ca.key -out ca.crt -subj '/CN=MyDemoCA'
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes -keyout cert.key -out cert.crt -CA ca.crt -CAkey ca.key -subj "/CN=minio" -addext "subjectAltName=DNS:minio"
