module github.com/markus-wa/demoinfocs-golang/v2

require (
	github.com/dustin/go-heatmap v0.0.0-20180603032536-b89dbd73785a
	github.com/gogo/protobuf v1.3.2
	github.com/golang/geo v0.0.0-20210211234256-740aa86cb551
	github.com/llgcode/draw2d v0.0.0-20200930101115-bfaf5d914d1e
	github.com/markus-wa/go-unassert v0.1.2
	github.com/markus-wa/gobitread v0.2.3
	github.com/markus-wa/godispatch v1.4.1
	github.com/markus-wa/ice-cipher-go v0.0.0-20220126215401-a6adadccc817
	github.com/markus-wa/quickhull-go/v2 v2.1.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.1.0 // indirect
	golang.org/x/image v0.0.0-20180708004352-c73c2afc3b81 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/dustin/go-heatmap => github.com/markus-wa/go-heatmap v1.0.0

replace github.com/stretchr/testify => github.com/stretchr/testify v1.6.1

go 1.18
