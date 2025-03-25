package global

import (
	"chainget/pkg"
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

var (
	Viper *viper.Viper //读取配置
)

func InitConfig() {
	Viper = viper.New()
	cwd, _ := os.Getwd()
	Viper.SetConfigFile(fmt.Sprintf("%s/%s", cwd, "config.yaml"))
	if err := Viper.ReadInConfig(); err != nil {
		log.Fatal("❌ 读取配置失败:", err)
	}
	Viper.Set("key.privateKey", GetPrivateKey())

	log.Println("✔️ 初始化配置成功")
}

func GetPrivateKey() string {
	pwd := os.Getenv("PASSWORD")
	if pwd == "" {
		log.Fatal("❌ 获取密码失败！！获取私钥失败")
	}
	decrypted, err := pkg.DecryptAESCBC(Viper.GetString("key.privateKey"), []byte(pwd))
	if err != nil {
		log.Fatalf("❌ 私钥解密失败: %v", err)
	}
	return string(decrypted)
}
