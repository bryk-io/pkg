# OpenAPI Application

Sample application that uses the [OpenAPI](<https://www.openapis.org>) specification
to define the interface of a web service. With the schema, you can then generate
a client and server implementation using ogen.

## Usage

* Install `ogen`

```bash
go install -v github.com/ogen-go/ogen/cmd/ogen@latest
```

* Generate the client and server implementation from your schema file

```bash
ogen -target petstore -clean petstore.yml
```

* Write a service implementation that conforms to the generated interface

```go
type ServiceOperator struct {}

func (svc *ServiceOperator) AddPet(ctx context.Context, req *api.Pet) (*api.Pet, error) {
  return nil, errors.New("not implemented")
}

// ...other methods...
```

* Run a server using your service implementation

```go
// prepare service operator
operator := newOperator()
svc, _ := api.NewServer(operator)

// prepare server instance
srv, _ := http.NewServer(
  http.WithHandler(svc), // use service operator as handler
  http.WithPort(8080),
  http.WithMiddleware(mwRecovery.Handler()),
  http.WithMiddleware(mwGzip.Handler(5)),
  http.WithMiddleware(mwLog.Handler(log, nil)),
)

// start server and listen for connections
log.Info("server ready")
return srv.Start()
```
