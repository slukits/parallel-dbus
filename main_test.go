package main

import (
	"fmt"
	"testing"

	. "github.com/slukits/gounit"
)

type RequestHandler struct{ Suite }

func (s *RequestHandler) SetUp(t *T) { t.Parallel() }

func (s *RequestHandler) Prints_help_if_no_sub_command_given(t *T) {
	got := ""
	handleRequest(mckArgs(mckPrint(t, &Env{}, &got)))
	t.Contains(got, help)
}

func (s *RequestHandler) Fails_on_unknown_sub_command(t *T) {
	expPnc, expErr, subCmd := "fatal mock panic", "", "unknown"
	defer func() {
		t.Log(expErr)
		t.Eq(expPnc, recover().(string))
		t.Contains(expErr, fmt.Sprintf(subErr, subCmd))
	}()
	handleRequest(mckArgs(
		mckFatal(t, &Env{}, expPnc, &expErr), subCmd))
}

func TestRequestHandler(t *testing.T) {
	t.Parallel()
	Run(&RequestHandler{}, t)
}
