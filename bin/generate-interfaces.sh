#!/bin/bash

set -e

go generate
go generate ./sendtables
