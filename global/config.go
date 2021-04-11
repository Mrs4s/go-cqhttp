package global

// AccountToken 存储AccountToken供登录使用
var AccountToken []byte

// PasswordHash 存储QQ密码哈希供登录使用
var PasswordHash [16]byte

/*
// GetCurrentPath 预留,获取当前目录地址
func GetCurrentPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	fpath, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		// fpath = strings.Replace(fpath, "\\", "/", -1)
		fpath = strings.ReplaceAll(fpath, "\\", "/")
	}
	i := strings.LastIndex(fpath, "/")
	if i < 0 {
		return "", errors.New("system/path_error,Can't find '/' or '\\'")
	}
	return fpath[0 : i+1], nil
}
*/
