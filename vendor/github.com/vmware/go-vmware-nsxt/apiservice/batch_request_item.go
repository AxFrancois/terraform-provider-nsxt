/* Copyright © 2017 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: BSD-2-Clause

   Generated by: https://github.com/swagger-api/swagger-codegen.git */

package apiservice

type BatchRequestItem struct {
	Body *interface{} `json:"body,omitempty"`

	// http method type
	Method string `json:"method"`

	// relative uri (path and args), of the call including resource id (if this is a POST/DELETE), exclude hostname and port and prefix, exploded form of parameters
	Uri string `json:"uri"`
}