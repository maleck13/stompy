package stompy

//A connection error indicates a problem with the network or socket
type ConnectionError string

//A Server error is something unexpected from the server
type ServerError string

//A bad frame means a frame was received but could not be parsed correctly
type BadFrameError string

//A generic error indicating unexpected state in the client
type ClientError string

//Version mis match
type VersionError string

func (ce ConnectionError) Error() string {
	return "unexpected connection error" + " : " + string(ce)
}

func (ce ServerError) Error() string {
	return "error returned from the server " + " : " + string(ce)
}

func (be BadFrameError) Error() string {
	return "bad frame recieved from server : " + string(be)
}

func (ce ClientError) Error() string {
	return "error occurred using the client : " + string(ce)
}

func (ve VersionError) Error() string {
	return "version error : " + string(ve)
}
