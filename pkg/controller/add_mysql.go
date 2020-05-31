package controller

import (
	"github.com/woohhan/sample-mysql-operator/pkg/controller/mysql"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, mysql.Add)
}
