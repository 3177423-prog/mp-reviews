package clients

import (
	"context"
	"io"
	"mime"
	"time"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
	"github.com/imroc/req/v3"
	"github.com/ogen-go/ogen/uri"
	"github.com/ogen-go/ogen/validate"

	"auto_answer/internal/domain"
)

// *** Работа с отзывами ***
// * Получить список отзывов *
//
// Доступно только для продавцов с подпиской Premium Plus.
// Вы можете оставить обратную связь по этому методу в комментариях к обсуждению в сообществе разработчиков Ozon for dev.
func (c *Client) ReviewList(ctx context.Context, request *domain.ReviewListRequest) (*domain.ReviewListResponse, error) {
	const MAX_REPEAT int16 = 5

	rc := req.C().
		SetCommonHeaderNonCanonical("Client-Id", c.oz.ClientID).
		SetCommonHeaderNonCanonical("Api-Key", c.oz.ApiKey)

	u := uri.Clone(c.requestURL(ctx))
	var pathParts [5]string
	pathParts[0] = "/v1/review/list"
	uri.AddPathParts(u, pathParts[:]...)

	r := rc.R()
	countRepeat := int16(1)
	for {
		if err := encodeReviewListRequest(request, r); err != nil {
			return nil, errors.Wrap(err, "encode request")
		}

		resp, err := r.Post(u.String())

		if err != nil {
			return nil, errors.Wrap(err, "do request")
		}
		defer resp.Body.Close()

		result, err, repeat := decodeReviewListResponse(resp)
		if err != nil {
			if repeat {
				countRepeat++
				if countRepeat < MAX_REPEAT {
					time.Sleep(10 * time.Second)
					continue
				}
			}

			return nil, errors.Wrap(err, "decode response")
		}

		return result, nil
	}
}

func encodeReviewListRequest(req *domain.ReviewListRequest, r *req.Request) error {
	const contentType = "application/json"
	e := new(jx.Encoder)
	{
		req.Encode(e)
	}
	encoded := e.Bytes()

	r.SetBodyBytes(encoded).SetContentType(contentType)

	return nil
}

func decodeReviewListResponse(resp *req.Response) (*domain.ReviewListResponse, error, bool) {
	ct, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, errors.Wrap(err, "parse media type"), false
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err, true
	}

	switch resp.StatusCode {
	case 200:
		switch ct {
		case "application/json":
			d := jx.DecodeBytes(buf)

			var response domain.ReviewListResponse
			if err := func() error {
				if err := response.Decode(d); err != nil {
					return err
				}
				if err := d.Skip(); err != io.EOF {
					return errors.New("unexpected trailing data")
				}
				return nil
			}(); err != nil {
				err = &DecodeBodyError{
					Status:      resp.Status,
					ContentType: ct,
					Body:        buf,
					Err:         err,
				}
				return nil, err, true
			}
			return &response, nil, false
		default:
			return nil, validate.InvalidContentType(ct), false
		}
	case 400, 401, 403, 423:
		err = &DecodeBodyError{
			Status:      resp.Status,
			ContentType: ct,
			Body:        buf,
			Err:         nil,
		}
		return nil, err, false
	case 404, 409, 429, 500:
		err = &DecodeBodyError{
			Status:      resp.Status,
			ContentType: ct,
			Body:        buf,
			Err:         nil,
		}
		return nil, err, true
	}

	return nil, validate.UnexpectedStatusCode(resp.StatusCode), true
}
