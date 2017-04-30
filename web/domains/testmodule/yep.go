// Copyright 2016 NDP Systèmes. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testmodule

import (
	// Import models definition of this module
	_ "github.com/npiganeau/yep-base/web/domains/testmodule/defs"
	"github.com/npiganeau/yep/yep/server"
)

// Module data declaration
const (
	MODULE_NAME string = "test_module"
)

func init() {
	declareMethods()
	server.RegisterModule(&server.Module{
		Name:     MODULE_NAME,
		PostInit: func() {},
	})
}
