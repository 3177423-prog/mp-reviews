package clients

import (
	"context"
	"io"
	"mime"
	"time"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
	"github.com/imroc/req/v3"
	"github.com/ogen-go/ogen/conv"
	"github.com/ogen-go/ogen/uri"
	"github.com/ogen-go/ogen/validate"

	"auto_answer/internal/domain"
	"auto_answer/internal/parameters"
)

// *** Общение с покупателями ***
// ** Отзывы **
// * Список отзывов *
//
// Метод предоставляет список отзывов по заданным фильтрам. Вы можете:
// - получить данные обработанных и необработанных отзывов
// - сортировать отзывы по дате
// - настроить пагинацию и количество отзывов в ответе
func (c *Client) WbFeedbackList(ctx context.Context, params *parameters.WbFeedbackList) (*domain.WbFeedbackResponse, error) {
	const MAX_REPEAT int16 = 5

	rc := req.C().
		SetCommonBearerAuthToken(c.wb.AccessToken)

	u := uri.Clone(c.requestURL(ctx))
	var pathParts [5]string
	pathParts[0] = "/api/v1/feedbacks"
	uri.AddPathParts(u, pathParts[:]...)

	q := make(map[string]string)
	{
		// Encode "isAnswered" parameter.
		q["isAnswered"] = conv.BoolToString(params.IsAnswered)
	}
	{
		// Encode "nmId" parameter.
		if val, ok := params.NmID.Get(); ok {
			q["nmId"] = conv.Int64ToString(val)
		}
	}
	{
		// Encode "take" parameter.
		if params.Take < 1 || params.Take > 5000 {
			return nil, errors.New("параметр \"take\" должен быть в пределах от 1 до 5 000")
		}
		q["take"] = conv.Uint32ToString(params.Take)
	}
	{
		// Encode "skip" parameter.
		if params.Skip > 199990 {
			return nil, errors.New("параметр \"skip\" должен быть в пределах от 1 до 199 990")
		}
		q["skip"] = conv.Uint32ToString(params.Skip)
	}
	{
		// Encode "order" parameter.
		if val, ok := params.Order.Get(); ok {
			q["order"] = conv.StringToString(string(val))
		}
	}
	{
		// Encode "dateFrom" parameter.
		if val, ok := params.DateFrom.Get(); ok {
			q["dateFrom"] = conv.Int64ToString(val.UnixMilli())
		}
	}
	{
		// Encode "dateTo" parameter.
		if val, ok := params.DateTo.Get(); ok {
			q["dateTo"] = conv.Int64ToString(val.UnixMilli())
		}
	}

	r := rc.R().SetQueryParams(q)
	countRepeat := int16(1)
	for {
		resp, err := r.Get(u.String())

		if err != nil {
			return nil, errors.Wrap(err, "do request")
		}
		defer resp.Body.Close()

		result, err, repeat := decodeWbFeedbackListResponse(resp)
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

func decodeWbFeedbackListResponse(resp *req.Response) (*domain.WbFeedbackResponse, error, bool) {
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

			var response domain.WbFeedbackResponse
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
