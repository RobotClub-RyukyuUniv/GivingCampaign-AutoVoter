// Package config は投票のたびに同じ答えになる項目を保存する。
//
// 氏名・生年月日・電話番号などの個人情報は Cross Campus のプロフィールから毎回取得するので、
// ここには保存しない。
package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// 応援メッセージの既定値（告知文に「コメント欄に「〜」と記入」があればそちらを提案する）
	Message string `yaml:"message,omitempty"`
	// 支援者として、お名前・大学との関係性の一般公開・支援先団体への共有に同意するか
	SupporterPublicationConsent *bool `yaml:"supporter_publication_consent,omitempty"`
	// 卒業/修了見込み年度（例: 2027）
	GraduationFiscalYear int `yaml:"graduation_fiscal_year,omitempty"`
	// 投票先団体との関係性: target_group_member / prior_connection / campaign_connection / support_project / other
	VoteReason string `yaml:"vote_reason,omitempty"`
	// どのようにしてGiving Campaignを知りましたか（複数可）:
	// student / other_individual / university_official_announcement / giving_campaign_official_announcement / media / other
	Sources []string `yaml:"sources,omitempty"`
	// 以下は Cross Campus のプロフィールに無い場合だけ使う
	Grade      string `yaml:"grade,omitempty"` // bachelor_1..6 / master_1..2 / doctor_1..3
	Faculty    string `yaml:"faculty,omitempty"`
	Department string `yaml:"department,omitempty"`
}

// DefaultPath は OS の設定ディレクトリ配下の gcvote/config.yaml。
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gcvote", "config.yaml"), nil
}

// Load は設定を読む。ファイルが無ければ空の設定を返す。
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func Save(path string, c *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
