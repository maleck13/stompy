package stompy

type HEADERS map[string]string

type InvalidHeader struct {
	message string
}

func (ih *InvalidHeader) Error() string {
	return ih.message
}

func connectionHeaders(opts ClientOpts) (HEADERS, error) {
	headers := HEADERS{}
	if opts.User != "" && opts.PassCode != "" {
		headers["login"] = opts.User
		headers["passcode"] = opts.PassCode
	}
	if "" == opts.Version {
		return nil, &InvalidHeader{"missing header accept-version ensure Version set in opts "}
	}
	headers["accept-version"] = opts.Version

	if "" == opts.Vhost {
		return nil, &InvalidHeader{"missing header host ensure Vhost set in opts"}
	}
	headers["host"] = opts.Vhost
	return headers, nil
}

func sendHeaders(dest, contentType string, addedHeaders HEADERS) HEADERS {
	headers := HEADERS{}
	headers["content-type"] = contentType
	headers["destination"] = dest
	if nil == addedHeaders {
		return headers
	}
	for k, v := range addedHeaders {
		headers[k] = v
	}
	return headers
}


func subscribeHeaders(id,dest string )HEADERS{
   	headers := HEADERS{}
   	headers["id"] = id
	headers["destination"] = dest
	return headers
}
