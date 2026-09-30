package domain

import (
	"math/bits"
	"strconv"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
	"github.com/ogen-go/ogen/validate"
)

// Данные для доступа к Озон
type AuthenticationOzonApi struct {
	ClientID string `json:"client_id"` // Идентификатор клиента
	ApiKey   string `json:"api_key"`   // Ключ API
}

var jsonFieldsNameOfAuthenticationOzonApi = [2]string{
	0: "client_id",
	1: "api_key",
}

func (s *AuthenticationOzonApi) Encode(e *jx.Encoder) {
	e.ObjStart()
	s.encodeFields(e)
	e.ObjEnd()
}

func (s *AuthenticationOzonApi) encodeFields(e *jx.Encoder) {
	{
		e.FieldStart("client_id")
		e.Str(s.ClientID)
	}
	{
		e.FieldStart("api_key")
		e.Str(s.ApiKey)
	}
}

func (s *AuthenticationOzonApi) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode AuthenticationOzonApi to nil")
	}
	var requiredBitSet [1]uint8

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "client_id":
			requiredBitSet[0] |= 1 << 0
			if err := func() error {
				v, err := d.Str()
				s.ClientID = string(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"client_id\"")
			}
		case "api_key":
			requiredBitSet[0] |= 1 << 1
			if err := func() error {
				v, err := d.Str()
				s.ApiKey = string(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"api_key\"")
			}
		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode AuthenticationOzonApi")
	}

	var failures []validate.FieldError
	for i, mask := range [1]uint8{
		0b00000011,
	} {
		if result := (requiredBitSet[i] & mask) ^ mask; result != 0 {
			missed := bits.OnesCount8(result)
			for bitN := 0; bitN < missed; bitN++ {
				bitIdx := bits.TrailingZeros8(result)
				fieldIdx := i*8 + bitIdx
				var name string
				if fieldIdx < len(jsonFieldsNameOfAuthenticationOzonApi) {
					name = jsonFieldsNameOfAuthenticationOzonApi[fieldIdx]
				} else {
					name = strconv.Itoa(fieldIdx)
				}
				failures = append(failures, validate.FieldError{
					Name:  name,
					Error: validate.ErrFieldRequired,
				})
				// Reset bit.
				result &^= 1 << bitIdx
			}
		}
	}
	if len(failures) > 0 {
		return &validate.Error{Fields: failures}
	}

	return nil
}

func (s *AuthenticationOzonApi) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *AuthenticationOzonApi) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (s *AuthenticationOzonApi) Validate() error {
	if s == nil {
		return validate.ErrNilPointer
	}

	var failures []validate.FieldError
	if err := func() error {
		if err := (validate.String{
			MinLength:    1,
			MinLengthSet: true,
			MaxLength:    255,
			MaxLengthSet: true,
			Email:        false,
			Hostname:     false,
		}).Validate(string(s.ClientID)); err != nil {
			return errors.Wrap(err, "string")
		}
		return nil
	}(); err != nil {
		failures = append(failures, validate.FieldError{
			Name:  "client_id",
			Error: err,
		})
	}
	if err := func() error {
		if err := (validate.String{
			MinLength:    1,
			MinLengthSet: true,
			MaxLength:    255,
			MaxLengthSet: true,
			Email:        false,
			Hostname:     false,
			Regex:        regexMap["uuid"],
		}).Validate(string(s.ApiKey)); err != nil {
			return errors.Wrap(err, "string")
		}
		return nil
	}(); err != nil {
		failures = append(failures, validate.FieldError{
			Name:  "api_key",
			Error: err,
		})
	}
	if len(failures) > 0 {
		return &validate.Error{Fields: failures}
	}
	return nil
}
