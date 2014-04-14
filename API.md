API
===

### Prepare Upload

```bash
curl -i -X GET http://localhost:8080/api/getfilename?ext=png
HTTP/1.1 200 OK
Content-Length: 17
Content-Type: text/plain; charset=utf-8

1twm86kqk9z67.png
```

### Upload File

```bash
curl -i -X PUT --data-binary "@file.png" http://localhost:8080/1twm86kqk9z67.png
HTTP/1.1 100 Continue

HTTP/1.1 200 OK
Content-Length: 0
Content-Type: text/plain; charset=utf-8
```
