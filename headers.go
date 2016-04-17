package stompy

import "strings"

type StompHeaders map[string]string

type encoding struct {
	To   string
	From string
}

var encoders = map[string][]encoding{
	"1.1": []encoding{
		encoding{"\\\\", "\\"},
		encoding{"\\n", "\n"},
		encoding{"\\c", ":"},
	},
	"1.2": []encoding{
		encoding{"\\\\", "\\"},
		encoding{"\\n", "\n"},
		encoding{"\\c", ":"},
		encoding{"\\r", "\r"},
	},
}

type encoder interface {
	Encode(val string) string
}

type decoder interface {
	Decode(val string) string
}

type encoderDecoder interface {
	encoder
	decoder
}

type headerEncoderDecoder struct {
	version string
}

func (hd headerEncoderDecoder) Decode(val string) string {
	encodings := encoders[hd.version]
	for _, enc := range encodings {
		val = strings.Replace(val, enc.To, enc.From, -1)
	}
	return val
}

func (hd headerEncoderDecoder) Encode(val string) string {
	encodings := encoders[hd.version]
	for _, enc := range encodings {
		val = strings.Replace(val, enc.From, enc.To, -1)
	}
	return val
}

type InvalidHeader struct {
	message string
}

func (ih *InvalidHeader) Error() string {
	return ih.message
}

func connectionHeaders(opts ClientOpts) (StompHeaders, error) {
	headers := StompHeaders{}
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

func sendHeaders(dest, contentType string, addedHeaders StompHeaders) StompHeaders {
	headers := StompHeaders{}
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

func subscribeHeaders(id, dest string, addedHeaders StompHeaders) StompHeaders {
	headers := StompHeaders{}
	headers["id"] = id
	headers["destination"] = dest
	if nil == addedHeaders {
		return headers
	}
	for k, v := range addedHeaders {
		headers[k] = v
	}
	return headers
}

func transactionHeaders(transactionId string, addedHeaders StompHeaders) StompHeaders {
	headers := StompHeaders{}
	headers["transaction"] = transactionId
	if nil == addedHeaders {
		return headers
	}
	for k, v := range addedHeaders {
		headers[k] = v
	}
	return headers

}

func unSubscribeHeaders(subId string, addedHeaders StompHeaders) StompHeaders {
	headers := StompHeaders{}
	headers["id"] = subId
	if nil == addedHeaders {
		return headers
	}
	for k, v := range addedHeaders {
		headers[k] = v
	}
	return headers
}

func nackHeaders(subId, messageId, transId string) StompHeaders {
	headers := StompHeaders{}
	headers["subscription"] = subId
	headers["message-id"] = messageId
	if "" != transId {
		headers["transaction"] = transId
	}
	return headers
}

func ackHeaders(subId, messageId, transId string) StompHeaders {
	headers := StompHeaders{}
	headers["subscription"] = subId
	headers["message-id"] = messageId
	if "" != transId {
		headers["transaction"] = transId
	}
	return headers
}
