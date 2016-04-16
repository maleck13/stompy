package stompy

type ConnectionError string
type ServerError string
type BadFrameError string
type ClientError string
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
