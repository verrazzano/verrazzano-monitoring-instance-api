// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateConfigMapKeyName(t *testing.T) {
	//valid
	var keyName = "foo"
	e := ValidateConfigMapKeyName(keyName)
	if e != nil {
		t.Logf("key name [%s]: expected valid, but returned invalid: %s", keyName, e)
		t.Fail()
	}

	//valid
	keyName = "foo.bar"
	e = ValidateConfigMapKeyName(keyName)
	if e != nil {
		t.Logf("key name [%s]: expected valid, but returned invalid: %s", keyName, e)
		t.Fail()
	}

	//not valid -- too short
	keyName = ""
	e = ValidateConfigMapKeyName(keyName)
	if e == nil {
		t.Logf("key name [%s]: expected invalid, but returned valid", keyName)
		t.Fail()
	}

	//not valid -- too long
	b := make([]rune, 254)
	for i := range b {
		b[i] = 'a'
	}
	keyName = string(b)
	e = ValidateConfigMapKeyName(keyName)
	if e == nil {
		t.Logf("key name [%s]: expected invalid, but returned valid", keyName)
		t.Fail()
	}

	//not valid -- no slashes
	keyName = "foo/bar"
	e = ValidateConfigMapKeyName(keyName)
	if e == nil {
		t.Logf("key name [%s]: expected invalid, but returned valid", keyName)
		t.Fail()
	}
}

func verifyStatus(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int) {
	if status := rr.Code; status != expectedStatus {
		t.Errorf("handler returned wrong status code: got %v want %v", status, expectedStatus)
	}
}

func verify(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedContent string) {
	verifyStatus(t, rr, expectedStatus)
	// We can't stop callers from passing the empty string, so we special case it as an
	// explicit test because every string contains the 0-length string as a substring.
	trimmedBody := strings.Trim(rr.Body.String(), " \t\n\r")
	if len(expectedContent) == 0 && len(trimmedBody) != 0 {
		t.Errorf("explicit check for empty body failed; use verifyStatus() to check just the status code. "+
			"Trimmed body content: '%v'", trimmedBody)
	}
	if !strings.Contains(rr.Body.String(), expectedContent) {
		t.Errorf("handler returned unexpected body: got '%v' cannot find '%v'", rr.Body.String(), expectedContent)
	}
}
