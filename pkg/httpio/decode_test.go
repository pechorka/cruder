package httpio_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pechorka/cruder/pkg/httpio"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	t.Run("query params only", func(t *testing.T) {
		type fullName struct {
			First  string  `query:"first"`
			Last   string  `query:"last"`
			Middle *string `query:"middle"`
		}
		type input struct {
			Name   fullName `query:"name"`
			Age    int      `query:"age"`
			Banned bool     `query:"banned"`
			Income uint     `query:"income"`
		}

		r := httptest.NewRequest("GET", "/?name.first=John&name.last=Doe&age=30&banned=true&income=100000&name.middle=Middle", nil)

		var v input
		err := httpio.Unmarshal(r, &v)
		require.NoError(t, err)

		require.Equal(t, "John", v.Name.First)
		require.Equal(t, "Doe", v.Name.Last)
		require.Equal(t, "Middle", *v.Name.Middle)
		require.Equal(t, 30, v.Age)
		require.Equal(t, true, v.Banned)
		require.Equal(t, uint(100000), v.Income)
	})

	t.Run("json and query params", func(t *testing.T) {
		type fullName struct {
			First string `query:"first"`
			Last  string `query:"last"`
		}
		type input struct {
			Name      fullName `query:"name"`
			Age       int      `query:"age"`
			Banned    bool     `query:"banned"`
			Income    uint     `query:"income"`
			AppConfig struct {
				Host string `json:"host"`
				Port int    `json:"port"`
			} `json:"app_config"`
		}

		body := `{"app_config":{"host":"localhost","port":8080}}`

		r := httptest.NewRequest("POST", "/?name.first=John&name.last=Doe&age=30&banned=true&income=100000", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		var v input
		err := httpio.Unmarshal(r, &v)
		require.NoError(t, err)

		require.Equal(t, "John", v.Name.First)
		require.Equal(t, "Doe", v.Name.Last)
		require.Equal(t, 30, v.Age)
		require.Equal(t, true, v.Banned)
		require.Equal(t, uint(100000), v.Income)
		require.Equal(t, "localhost", v.AppConfig.Host)
		require.Equal(t, 8080, v.AppConfig.Port)
	})
}

func BenchmarkUnmarshal(b *testing.B) {
	type fullName struct {
		First string `query:"first"`
		Last  string `query:"last"`
	}
	type input struct {
		Name    fullName `query:"name"`
		Age     int      `query:"age"`
		Banned  bool     `query:"banned"`
		Income  uint     `query:"income"`
		Field1  fullName `query:"field1"`
		Field2  fullName `query:"field2"`
		Field3  fullName `query:"field3"`
		Field4  fullName `query:"field4"`
		Field5  fullName `query:"field5"`
		Field6  fullName `query:"field6"`
		Field7  fullName `query:"field7"`
		Field8  fullName `query:"field8"`
		Field9  fullName `query:"field9"`
		Field10 fullName `query:"field10"`
		Field11 struct {
			Field1  fullName `query:"field1"`
			Field2  fullName `query:"field2"`
			Field3  fullName `query:"field3"`
			Field4  fullName `query:"field4"`
			Field5  fullName `query:"field5"`
			Field6  fullName `query:"field6"`
			Field7  fullName `query:"field7"`
			Field8  fullName `query:"field8"`
			Field9  fullName `query:"field9"`
			Field10 fullName `query:"field10"`
			Field11 struct {
				Field1  fullName `query:"field1"`
				Field2  fullName `query:"field2"`
				Field3  fullName `query:"field3"`
				Field4  fullName `query:"field4"`
				Field5  fullName `query:"field5"`
				Field6  fullName `query:"field6"`
				Field7  fullName `query:"field7"`
				Field8  fullName `query:"field8"`
				Field9  fullName `query:"field9"`
				Field10 fullName `query:"field10"`
				Field11 struct {
					Field1  fullName `query:"field1"`
					Field2  fullName `query:"field2"`
					Field3  fullName `query:"field3"`
					Field4  fullName `query:"field4"`
					Field5  fullName `query:"field5"`
					Field6  fullName `query:"field6"`
					Field7  fullName `query:"field7"`
					Field8  fullName `query:"field8"`
					Field9  fullName `query:"field9"`
					Field10 fullName `query:"field10"`
					Field11 struct {
						Field1  fullName `query:"field1"`
						Field2  fullName `query:"field2"`
						Field3  fullName `query:"field3"`
						Field4  fullName `query:"field4"`
						Field5  fullName `query:"field5"`
						Field6  fullName `query:"field6"`
						Field7  fullName `query:"field7"`
						Field8  fullName `query:"field8"`
						Field9  fullName `query:"field9"`
						Field10 fullName `query:"field10"`
						Field11 struct {
							Field1  fullName `query:"field1"`
							Field2  fullName `query:"field2"`
							Field3  fullName `query:"field3"`
							Field4  fullName `query:"field4"`
							Field5  fullName `query:"field5"`
							Field6  fullName `query:"field6"`
							Field7  fullName `query:"field7"`
							Field8  fullName `query:"field8"`
							Field9  fullName `query:"field9"`
							Field10 fullName `query:"field10"`
							Field11 struct {
								Field1 fullName `query:"field1"`
							} `query:"field11"`
						} `query:"field11"`
					} `query:"field11"`
				} `query:"field11"`
			} `query:"field11"`
		} `query:"field11"`
	}

	r := httptest.NewRequest("GET", "/?name.first=John&name.last=Doe&age=30&banned=true&income=100000", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var v input
		err := httpio.Unmarshal(r, &v)
		require.NoError(b, err)
	}
}
