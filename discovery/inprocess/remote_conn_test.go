package inprocess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/mcontext"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestRemoteClientConnInvokeSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, BroadcastPath, r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Equal(t, "op-123", r.Header.Get(constant.OperationID))

		var req InvokeRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "messagegateway", req.Service)
		require.Equal(t, "/inprocess.Test/Echo", req.Method)
		require.Equal(t, "secret", req.Secret)

		var in wrapperspb.StringValue
		require.NoError(t, proto.Unmarshal(req.Request, &in))
		out, err := proto.Marshal(wrapperspb.String(in.Value + ":remote"))
		require.NoError(t, err)
		require.NoError(t, json.NewEncoder(w).Encode(InvokeResponse{Data: out}))
	}))
	defer server.Close()

	conn := newRemoteClientConn(server.URL, "messagegateway", "secret")
	ctx := mcontext.SetOperationID(context.Background(), "op-123")
	var reply wrapperspb.StringValue
	require.NoError(t, conn.Invoke(ctx, "/inprocess.Test/Echo", wrapperspb.String("hello"), &reply))
	require.Equal(t, "hello:remote", reply.Value)
}

func TestRemoteClientConnInvokeErrCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(InvokeResponse{
			ErrCode: 1004,
			ErrMsg:  "record not found",
			ErrDlt:  "user not exist",
		}))
	}))
	defer server.Close()

	conn := newRemoteClientConn(server.URL, "messagegateway", "secret")
	var reply wrapperspb.StringValue
	err := conn.Invoke(context.Background(), "/inprocess.Test/Echo", wrapperspb.String("hello"), &reply)
	require.Error(t, err)
	codeErr, ok := errs.Unwrap(err).(errs.CodeError)
	require.True(t, ok, "expected CodeError, got %T: %v", errs.Unwrap(err), err)
	require.Equal(t, 1004, codeErr.Code())
	require.Equal(t, "record not found", codeErr.Msg())
	require.Equal(t, "user not exist", codeErr.Detail())
}

func TestRemoteClientConnInvokeHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	conn := newRemoteClientConn(server.URL, "messagegateway", "secret")
	var reply wrapperspb.StringValue
	err := conn.Invoke(context.Background(), "/inprocess.Test/Echo", wrapperspb.String("hello"), &reply)
	require.Error(t, err)
	require.Contains(t, err.Error(), "status")
}
