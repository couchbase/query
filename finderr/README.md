## Finderror Utility
### Sections
    - CODE : error number
    - NAME : error name
    - DESCRIPTION : error description(the internal message )
    - CAUSE : steps that may have triggered the error to be raised.
    - ACTIONS: steps to take to mitigate the error( if possible )

### Build
```sh
cd finderr
```

```sh
go build .
```


### Usage:
```sh
./finderr [code-number/name]
```

For eg:
```sh
./finderr 1000
```

```sh
./finderr E_SERVICE_READONLY
```

### Internal
steps to add a new error, add an entry in:
- codes.go
- errormessages.go
- namecodemap.go