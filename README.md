- [🚀 Running](#-running)
  - [Without checking last *low\_stock*](#without-checking-last-low_stock)
  - [With checking last *low\_stock*](#with-checking-last-low_stock)
- [🛠️ Building](#️-building)
  - [On MacOS](#on-macos)
    - [for Windows](#for-windows)
    - [for MacOS (Apple Silicon)](#for-macos-apple-silicon)
    - [for MacOS (Intel)](#for-macos-intel)
    - [for Linux](#for-linux)


---

# 🚀 Running
## Without checking last *low_stock*
With default filenames (`etsy.csv` and `sklad.csv`)
```bash
# Windows (in cmd)
main.exe

# MacOS
./main
```

Or with custom `.csv` filenames
```bash
# Windows (in cmd)
main.exe -etsy=etsy_export.csv -money=napa2111.csv

# MacOS
./main -etsy=etsy_export.csv -money=napa2111.csv
```


## With checking last *low_stock*
> ols = _old low stock_

```bash
# Windows (in cmd)
main.exe -ols=low_stock.old.csv

# MacOS
./main -ols=low_stock.old.csv
```



# 🛠️ Building
> You have to have Go installed

## On MacOS
### for Windows
```bash
env GOOS=windows GOARCH=amd64 go build -o bin/main.exe main.go
```

### for MacOS (Apple Silicon)
```bash
env GOOS=darwin GOARCH=arm64 go build -o bin/main main.go
```

### for MacOS (Intel)
```bash
env GOOS=darwin GOARCH=amd64 go build -o bin/main main.go
```

### for Linux
```bash
env GOOS=linux GOARCH=amd64 go build -o bin/main main.go
```

