package helper

// TODO: Implement bad request and generic http interceptors
func ErrorPanic(err error) {
	if err != nil {
		panic(err)
	}
}
