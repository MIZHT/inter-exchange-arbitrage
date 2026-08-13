package core_config

type Exchange struct {
	Enabled    bool   `envconfig:"ENABLED"` // enabled нужно ли ???
	APIKey     string `envconfig:"API_KEY"`
	Secret     string `envconfig:"SECRET"`
	Passphrase string `envconfig:"PASSPHRASE"`
}
