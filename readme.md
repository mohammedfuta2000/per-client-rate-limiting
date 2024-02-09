# Per Client Rate Limiting

### Run the Application

```bash
$ go run .

```
### Test it

```bash
$ for i in {1..6}; do
  curl -i http://localhost:9000/ping
done
```