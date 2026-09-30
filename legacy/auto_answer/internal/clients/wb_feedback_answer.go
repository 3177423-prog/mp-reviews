package clients

import (
	"context"
	"io"
	"time"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
	"github.com/imroc/req/v3"
	"github.com/ogen-go/ogen/uri"
	"github.com/ogen-go/ogen/validate"

	"auto_answer/internal/domain"
)

// *** Общение с покупателями ***
// ** Отзывы **
// * Ответить на отзыв *
//
// Метод позволяет ответить на отзыв покупателя.
func (c *Client) WbFeedbackAnswer(ctx context.Context, request *domain.WbFeedbackAnswerRequest) error {
	const MAX_REPEAT int16 = 1

	rc := req.C().
		SetCommonBearerAuthToken(c.wb.AccessToken)

	u := uri.Clone(c.requestURL(ctx))
	var pathParts [5]string
	pathParts[0] = "/api/v1/feedbacks/answer"
	uri.AddPathParts(u, pathParts[:]...)

	r := rc.R()
	countRepeat := int16(1)
	for {
		if err := encodeWbFeedbackAnswerRequest(request, r); err != nil {
			return errors.Wrap(err, "encode request")
		}

		resp, err := r.Post(u.String())
		if err != nil {
			return errors.Wrap(err, "do request")
		}
		defer resp.Body.Close()

		err, repeat := decodeWbFeedbackAnswerResponse(resp)
		if err != nil {
			if repeat {
				countRepeat++
				if countRepeat < MAX_REPEAT {
					time.Sleep(10 * time.Second)
					continue
				}
			}

			return errors.Wrap(err, "decode response")
		}

		return nil
	}
}

func encodeWbFeedbackAnswerRequest(req *domain.WbFeedbackAnswerRequest, r *req.Request) error {
	const contentType = "application/json"
	e := new(jx.Encoder)
	{
		req.Encode(e)
	}
	encoded := e.Bytes()

	r.SetBodyBytes(encoded).SetContentType(contentType)

	return nil
}

func decodeWbFeedbackAnswerResponse(resp *req.Response) (error, bool) {
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return err, true
	}

	switch resp.StatusCode {
	case 204:
		return nil, false
	case 400, 401, 403, 423:
		err = &DecodeBodyError{
			Status: resp.Status,
			Body:   buf,
			Err:    nil,
		}
		return err, false
	case 404, 409, 429, 500:
		err = &DecodeBodyError{
			Status: resp.Status,
			Body:   buf,
			Err:    nil,
		}
		return err, true
	}

	return validate.UnexpectedStatusCode(resp.StatusCode), true
}
