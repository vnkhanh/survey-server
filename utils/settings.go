package utils

import (
	"encoding/json"
	"errors"
	"time"
)

type NullableInt struct {
	Set   bool
	Value *int
}

func (n *NullableInt) UnmarshalJSON(data []byte) error {
	n.Set = true
	// null
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	// số
	var v int
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}

func (n NullableInt) MarshalJSON() ([]byte, error) {
	if n.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.Value)
}

type FormSettings struct {
	MaxResponses     NullableInt `json:"max_responses,omitempty"`     // giới hạn tổng số lượt trả lời (nil = không giới hạn)
	CollectEmail     *bool       `json:"collect_email,omitempty"`     // yêu cầu nhập email
	ShowProgress     *bool       `json:"show_progress,omitempty"`     // hiển thị progress bar
	ShuffleQuestions *bool       `json:"shuffle_questions,omitempty"` // xáo trộn câu hỏi
	StartAt          *int64      `json:"start_at,omitempty"`          // thời điểm bắt đầu (unix seconds)
	ExpireAt         *int64      `json:"expire_at,omitempty"`         // thời điểm hết hạn (unix seconds)
	Language         string      `json:"language,omitempty"`          // ngôn ngữ hiển thị ("vi", "en")
}

// ValidateSettings với clamp cho MaxResponses
func ValidateSettings(s *FormSettings) error {
	if s == nil {
		return errors.New("settings rỗng")
	}
	if s.MaxResponses.Set && s.MaxResponses.Value != nil {
		if *s.MaxResponses.Value < 1 {
			v := 1
			s.MaxResponses.Value = &v
		}
	}
	if s.StartAt != nil && s.ExpireAt != nil && *s.ExpireAt <= *s.StartAt {
		return errors.New("expire_at phải lớn hơn start_at")
	}
	return nil
}

func ParseSettings(raw []byte) (*FormSettings, error) {
	if len(raw) == 0 {
		return &FormSettings{}, nil
	}
	var s FormSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, errors.New("settings không phải JSON hợp lệ")
	}
	if err := ValidateSettings(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func NormalizeSettings(s *FormSettings) *FormSettings {
	if s == nil {
		return &FormSettings{}
	}
	out := *s
	if out.Language == "" {
		out.Language = ""
	}
	return &out
}

func NormalizeSettingsJSON(s *FormSettings) (string, error) {
	if s == nil {
		s = &FormSettings{}
	}
	n := NormalizeSettings(s)
	b, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func MergeSettings(base *FormSettings, patch *FormSettings) *FormSettings {
	if base == nil {
		base = &FormSettings{}
	}
	if patch == nil {
		patch = &FormSettings{}
	}
	out := *base

	// Nếu client có gửi max_responses (dù null hay số) thì overwrite
	if patch.MaxResponses.Set {
		out.MaxResponses = patch.MaxResponses
	}
	if patch.CollectEmail != nil {
		out.CollectEmail = patch.CollectEmail
	}
	if patch.ShowProgress != nil {
		out.ShowProgress = patch.ShowProgress
	}
	if patch.ShuffleQuestions != nil {
		out.ShuffleQuestions = patch.ShuffleQuestions
	}
	if patch.StartAt != nil {
		out.StartAt = patch.StartAt
	}
	if patch.ExpireAt != nil {
		out.ExpireAt = patch.ExpireAt
	}
	if patch.Language != "" {
		out.Language = patch.Language
	}
	return &out
}

func NowUnix() int64 { return time.Now().Unix() }
