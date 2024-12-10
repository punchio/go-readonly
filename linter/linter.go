package linter

import (
	"fmt"
	"github.com/golangci/plugin-module-register/register"
	"github.com/punchio/go-readonly"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

func init() {
	register.Plugin("readonly", New)
}

func New(setting any) (register.LinterPlugin, error) {
	return &Plugin{}, nil
}

type Plugin struct {
}

func (p *Plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	ana := readonly.NewAnalyzer()
	r := ana.Run
	initialized := false
	ana.Run = func(pass *analysis.Pass) (interface{}, error) {
		if !initialized {
			initialized = true
			cfg := &packages.Config{
				Mode:  packages.NeedName | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule,
				Dir:   pass.Module.Path, // 指定项目目录
				Tests: false,            // 不包含测试文件
			}

			// 加载所有 package
			pkgs, err := packages.Load(cfg, "./...")
			if err != nil {
				return nil, fmt.Errorf("loading packages fail, err: %v", err)
			}
			readonly.Setup(pkgs)
		}
		return r(pass)
	}
	return []*analysis.Analyzer{ana}, nil
}

func (p *Plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
