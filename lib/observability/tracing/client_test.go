// Copyright 2022 Gravitational, Inc
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

package tracing

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	otlp "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRotatingFileClient(t *testing.T) {
	dir := t.TempDir()

	client, err := NewRotatingFileClient(dir, 10)
	require.NoError(t, err)

	// verify that creating the client creates a file
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// create a span to test with
	span := &otlp.ResourceSpans{
		Resource: &resourcev1.Resource{
			Attributes: []*commonv1.KeyValue{
				{
					Key: "test",
					Value: &commonv1.AnyValue{
						Value: &commonv1.AnyValue_IntValue{
							IntValue: 0,
						},
					},
				},
			},
		},
		ScopeSpans: []*otlp.ScopeSpans{
			{
				Spans: []*otlp.Span{
					{
						TraceId:           []byte{1, 2, 3, 4},
						SpanId:            []byte{5, 6, 7, 8},
						TraceState:        "",
						ParentSpanId:      []byte{9, 10, 11, 12},
						Name:              "test",
						Kind:              otlp.Span_SPAN_KIND_CLIENT,
						StartTimeUnixNano: uint64(time.Now().Add(-1 * time.Minute).Unix()),
						EndTimeUnixNano:   uint64(time.Now().Unix()),
						Attributes: []*commonv1.KeyValue{
							{
								Key: "test",
								Value: &commonv1.AnyValue{
									Value: &commonv1.AnyValue_IntValue{
										IntValue: 0,
									},
								},
							},
						},
						Status: &otlp.Status{
							Message: "success!",
							Code:    otlp.Status_STATUS_CODE_OK,
						},
					},
				},
			},
		},
	}

	// upload spans a bunch of spans
	testSpans := []*otlp.ResourceSpans{span, span, span}
	for i := 0; i < 10; i++ {
		require.NoError(t, client.UploadTraces(context.Background(), testSpans))
	}

	// stop the client to close and flush the files
	require.NoError(t, client.Stop(context.Background()))

	// get the names of all the files created and verify that files were rotated
	entries, err = os.ReadDir(dir)
	require.NoError(t, err)
	// the +1 here is because there should be an empty active file waiting for more writes
	require.Len(t, entries, 10*len(testSpans)+1)

	// read in all the spans that we just exported
	var spans []*otlp.ResourceSpans
	for _, entry := range entries {
		spans = append(spans, readFileTraces(t, filepath.Join(dir, entry.Name()))...)
	}

	// ensure that the number read matches the number of spans we uploaded
	require.Len(t, spans, 10*len(testSpans))

	// confirm that all spans are equivalent to our test span
	for _, fileSpan := range spans {
		require.Empty(t, cmp.Diff(span, fileSpan,
			cmpopts.IgnoreUnexported(
				otlp.ResourceSpans{},
				otlp.ScopeSpans{},
				otlp.Span{},
				otlp.Status{},
				resourcev1.Resource{},
				commonv1.KeyValue{},
				commonv1.AnyValue{},
			),
		))
	}
}

func readFileTraces(t *testing.T, filename string) []*otlp.ResourceSpans {
	t.Helper()

	f, err := os.Open(filename)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var spans []*otlp.ResourceSpans
	for scanner.Scan() {
		var span otlp.ResourceSpans
		require.NoError(t, protojson.Unmarshal(scanner.Bytes(), &span))

		spans = append(spans, &span)

	}

	require.NoError(t, scanner.Err())

	return spans
}
