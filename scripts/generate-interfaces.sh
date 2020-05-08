#!/bin/bash

set -e

go generate ./pkg/demoinfocs
go generate ./pkg/demoinfocs/sendtables
