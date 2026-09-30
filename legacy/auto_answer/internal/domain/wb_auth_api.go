package domain

import (
	"math/bits"
	"strconv"
	"time"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
	"github.com/ogen-go/ogen/json"
	"github.com/ogen-go/ogen/validate"
)

// Данные для доступа к Вайлдберриз
type AuthenticationWildberriesApi struct {
	TokenType      string    `json:"token_type"`      // Тип токена доступа
	AccessToken    string    `json:"access_token"`    // Токен доступа
	ExpiriesAccess time.Time `json:"expiries_access"` // Дата до которой токен действует
}

var jsonFieldsNameOfAuthenticationWildberriesApi = [3]string{
	0: "token_type",
	1: "access_token",
	2: "expiries_access",
}

func (s *AuthenticationWildberriesApi) Encode(e *jx.Encoder) {
	e.ObjStart()
	s.encodeFields(e)
	e.ObjEnd()
}

func (s *AuthenticationWildberriesApi) encodeFields(e *jx.Encoder) {
	{
		e.FieldStart("token_type")
		e.Str(s.TokenType)
	}
	{
		e.FieldStart("access_token")
		e.Str(s.AccessToken)
	}
	{
		e.FieldStart("expiries_access")
		json.EncodeDateTime(e, s.ExpiriesAccess)
	}
}

func (s *AuthenticationWildberriesApi) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode AuthenticationWildberriesApi to nil")
	}
	var requiredBitSet [1]uint8

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "token_type":
			requiredBitSet[0] |= 1 << 0
			if err := func() error {
				v, err := d.Str()
				s.TokenType = string(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"token_type\"")
			}
		case "access_token":
			requiredBitSet[0] |= 1 << 1
			if err := func() error {
				v, err := d.Str()
				s.AccessToken = string(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"access_token\"")
			}
		case "expiries_access":
			requiredBitSet[0] |= 1 << 2
			if err := func() error {
				v, err := json.DecodeDateTime(d)
				s.ExpiriesAccess = v
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"expiries_access\"")
			}
		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode AuthenticationWildberriesApi")
	}

	var failures []validate.FieldError
	for i, mask := range [1]uint8{
		0b00000111,
	} {
		if result := (requiredBitSet[i] & mask) ^ mask; result != 0 {
			missed := bits.OnesCount8(result)
			for bitN := 0; bitN < missed; bitN++ {
				bitIdx := bits.TrailingZeros8(result)
				fieldIdx := i*8 + bitIdx
				var name string
				if fieldIdx < len(jsonFieldsNameOfAuthenticationWildberriesApi) {
					name = jsonFieldsNameOfAuthenticationWildberriesApi[fieldIdx]
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

func (s *AuthenticationWildberriesApi) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *AuthenticationWildberriesApi) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (s *AuthenticationWildberriesApi) Validate() error {
	if s == nil {
		return validate.ErrNilPointer
	}

	var failures []validate.FieldError
	if err := func() error {
		if err := (validate.String{
			MinLength:    1,
			MinLengthSet: true,
			MaxLength:    0,
			MaxLengthSet: false,
			Email:        false,
			Hostname:     false,
		}).Validate(string(s.TokenType)); err != nil {
			return errors.Wrap(err, "string")
		}
		return nil
	}(); err != nil {
		failures = append(failures, validate.FieldError{
			Name:  "token_type",
			Error: err,
		})
	}
	if err := func() error {
		if err := (validate.String{
			MinLength:    1,
			MinLengthSet: true,
			MaxLength:    0,
			MaxLengthSet: false,
			Email:        false,
			Hostname:     false,
		}).Validate(string(s.AccessToken)); err != nil {
			return errors.Wrap(err, "string")
		}
		return nil
	}(); err != nil {
		failures = append(failures, validate.FieldError{
			Name:  "access_token",
			Error: err,
		})
	}
	if len(failures) > 0 {
		return &validate.Error{Fields: failures}
	}
	return nil
}
