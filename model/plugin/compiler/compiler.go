// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugincompiler

import (
	"fmt"
	"github.com/onosproject/onos-config/model"
	configmodule "github.com/onosproject/onos-config/model/plugin/module"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	_ "github.com/openconfig/gnmi/proto/gnmi" // gnmi
	_ "github.com/openconfig/goyang/pkg/yang" // yang
	_ "github.com/openconfig/ygot/genutil"    // genutil
	_ "github.com/openconfig/ygot/ygen"       // ygen
	_ "github.com/openconfig/ygot/ygot"       // ygot
	_ "github.com/openconfig/ygot/ytypes"     // ytypes
	_ "google.golang.org/protobuf/proto"      // proto
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

var log = logging.GetLogger("config-model", "compiler")

const (
	modelDir = "model"
	yangDir  = "yang"
)

const (
	mainTemplate   = "main.go.tpl"
	pluginTemplate = "plugin.go.tpl"
	modelTemplate  = "model.go.tpl"
)

const (
	mainFile   = "main.go"
	pluginFile = "plugin.go"
	modelFile  = "model.go"
)

const (
	defaultTemplatePath = "model/plugin/compiler/templates"
)

// TemplateInfo provides all the variables for templates
type TemplateInfo struct {
	Model configmodel.ModelInfo
}

// Config is a plugin compiler configuration
type Config struct {
	TemplatePath string
}

// NewPluginCompiler creates a new model plugin compiler
func NewPluginCompiler(module *configmodule.Module, config Config) *PluginCompiler {
	if config.TemplatePath == "" {
		config.TemplatePath = defaultTemplatePath
	}
	return &PluginCompiler{
		Config: config,
		module: module,
	}
}

// PluginCompiler is a model plugin compiler
type PluginCompiler struct {
	Config Config
	module *configmodule.Module
}

// CompilePlugin compiles a model plugin to the given path
func (c *PluginCompiler) CompilePlugin(model configmodel.ModelInfo, path string) error {
	log.Infof("Compiling ConfigModel '%s/%s' to '%s'", model.Name, model.Version, path)

	// Ensure the build directory exists
	c.createDir(c.module.Path)

	// Create the module files
	c.createDir(c.getPluginDir(model))
	if err := c.generateMain(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %s", model.Name, model.Version, err)
		return err
	}

	// Create the model plugin
	c.createDir(c.getModelDir(model))
	if err := c.generateConfigModel(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %s", model.Name, model.Version, err)
		return err
	}
	if err := c.generateModelPlugin(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %s", model.Name, model.Version, err)
		return err
	}

	// Generate the YANG bindings
	c.createDir(c.getYangDir(model))
	if err := c.copyFiles(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %s", model.Name, model.Version, err)
		return err
	}
	if err := c.generateYangBindings(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %s", model.Name, model.Version, err)
		return err
	}

	// Compile the plugin
	c.createDir(filepath.Dir(path))
	if err := c.compilePlugin(model, path); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %s", model.Name, model.Version, err)
		return err
	}

	// Clean up the build
	if err := c.cleanBuild(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %s", model.Name, model.Version, err)
		return err
	}
	return nil
}

func (c *PluginCompiler) getTemplateInfo(model configmodel.ModelInfo) (TemplateInfo, error) {
	return TemplateInfo{
		Model: model,
	}, nil
}

func (c *PluginCompiler) getPluginMod(model configmodel.ModelInfo) string {
	return fmt.Sprintf("github.com/onosproject/onos-config/models/%s", c.getSafeQualifiedName(model))
}

func (c *PluginCompiler) compilePlugin(model configmodel.ModelInfo, path string) error {
	log.Infof("Compiling plugin '%s'", path)
	log.Infof("go build -o %s -buildmode=plugin %s", path, c.getPluginMod(model))
	_, err := c.exec(c.module.Path, "go", "build", "-o", path, "-buildmode=plugin", c.getPluginMod(model))
	if err != nil {
		log.Errorf("Compiling plugin '%s' failed: %s", path, err)
		return err
	}
	return nil
}

func (c *PluginCompiler) exec(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=on", "CGO_ENABLED=1")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *PluginCompiler) cleanBuild(model configmodel.ModelInfo) error {
	if _, err := os.Stat(c.getPluginDir(model)); err == nil {
		return os.RemoveAll(c.getPluginDir(model))
	}
	return nil
}

func (c *PluginCompiler) copyFiles(model configmodel.ModelInfo) error {
	for _, file := range model.Files {
		if err := c.copyFile(model, file); err != nil {
			return err
		}
	}
	return nil
}

func (c *PluginCompiler) copyFile(model configmodel.ModelInfo, file configmodel.FileInfo) error {
	path := c.getYangPath(model, file)
	log.Debugf("Copying YANG module '%s' to '%s'", file.Path, path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := ioutil.WriteFile(path, file.Data, os.ModePerm)
		if err != nil {
			log.Errorf("Copying YANG module '%s' failed: %s", file.Path, err)
			return err
		}
	}
	return nil
}

func (c *PluginCompiler) generateYangBindings(model configmodel.ModelInfo) error {
	path := filepath.Join(c.getModelPath(model, "generated.go"))
	log.Debugf("Generating YANG bindings '%s'", path)
	args := []string{
		"run",
		"github.com/openconfig/ygot/generator",
		fmt.Sprintf("-path=%s/yang", c.getPluginDir(model)),
		fmt.Sprintf("-output_file=%s/model/generated.go", c.getPluginDir(model)),
		"-package_name=configmodel",
		"-generate_fakeroot",
	}

	for _, module := range model.Modules {
		args = append(args, module.File)
	}

	log.Infof("Run compilation in %s with go %s", c.getPluginDir(model), strings.Join(args, " "))
	cmd := exec.Command("go", args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Generating YANG bindings '%s' failed: %s", path, err)
		return err
	}
	return nil
}

func (c *PluginCompiler) getTemplatePath(name string) string {
	return filepath.Join(c.Config.TemplatePath, name)
}

func (c *PluginCompiler) generateMain(model configmodel.ModelInfo) error {
	info, err := c.getTemplateInfo(model)
	if err != nil {
		return err
	}
	return applyTemplate(mainTemplate, c.getTemplatePath(mainTemplate), c.getPluginPath(model, mainFile), info)
}

func (c *PluginCompiler) generateTemplate(model configmodel.ModelInfo, template, inPath, outPath string) error {
	log.Debugf("Generating '%s'", outPath)
	info, err := c.getTemplateInfo(model)
	if err != nil {
		log.Errorf("Generating '%s' failed: %s", outPath, err)
		return err
	}
	if err := applyTemplate(template, inPath, outPath, info); err != nil {
		log.Errorf("Generating '%s' failed: %s", outPath, err)
		return err
	}
	return nil
}

func (c *PluginCompiler) generateModelPlugin(model configmodel.ModelInfo) error {
	return c.generateTemplate(model, pluginTemplate, c.getTemplatePath(pluginTemplate), c.getModelPath(model, pluginFile))
}

func (c *PluginCompiler) generateConfigModel(model configmodel.ModelInfo) error {
	return c.generateTemplate(model, modelTemplate, c.getTemplatePath(modelTemplate), c.getModelPath(model, modelFile))
}

func (c *PluginCompiler) getPluginDir(model configmodel.ModelInfo) string {
	return filepath.Join(c.module.Path, "models", c.getSafeQualifiedName(model))
}

func (c *PluginCompiler) getPluginPath(model configmodel.ModelInfo, name string) string {
	return filepath.Join(c.getPluginDir(model), name)
}

func (c *PluginCompiler) getModelDir(model configmodel.ModelInfo) string {
	return filepath.Join(c.getPluginDir(model), modelDir)
}

func (c *PluginCompiler) getModelPath(model configmodel.ModelInfo, name string) string {
	return filepath.Join(c.getModelDir(model), name)
}

func (c *PluginCompiler) getYangDir(model configmodel.ModelInfo) string {
	return filepath.Join(c.getPluginDir(model), yangDir)
}

func (c *PluginCompiler) getYangPath(model configmodel.ModelInfo, file configmodel.FileInfo) string {
	return filepath.Join(c.getYangDir(model), filepath.Base(file.Path))
}

func (c *PluginCompiler) getSafeQualifiedName(model configmodel.ModelInfo) string {
	return strings.ReplaceAll(fmt.Sprintf("%s_%s", model.Name, model.Version), ".", "_")
}

func (c *PluginCompiler) createDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Debugf("Creating '%s'", dir)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Errorf("Creating '%s' failed: %s", dir, err)
		}
	}
}

func applyTemplate(name, tplPath, outPath string, data TemplateInfo) error {
	var funcs template.FuncMap = map[string]interface{}{
		"quote": func(value interface{}) string {
			return fmt.Sprintf("\"%s\"", value)
		},
		"replace": func(search, replace string, value interface{}) string {
			return strings.ReplaceAll(fmt.Sprint(value), search, replace)
		},
	}

	tpl, err := template.New(name).
		Funcs(funcs).
		ParseFiles(tplPath)
	if err != nil {
		return err
	}

	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tpl.Execute(file, data)
}
