# prusa_proxy

Just small proxy that allow user control printer via REST API without necessary access to digest - for example when using Grafana

### Operations

#### Pause

```
curl --location 'http://localhost:31100/pause' \
--header 'Content-Type: text/plain' \
--data '{
    "ip": "<IP address of printer>"
}'
```

#### Resume

```
curl --location 'http://localhost:31100/resume' \
--header 'Content-Type: text/plain' \
--data '{
    "ip": "<IP address of printer>"
}'
```

#### Stop

```
curl --location 'http://localhost:31100/stop' \
--header 'Content-Type: text/plain' \
--data '{
    "ip": "<IP address of printer>"
}'
```
