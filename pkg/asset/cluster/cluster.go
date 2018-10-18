package cluster

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/installer/data"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/terraform"
	"github.com/openshift/installer/pkg/types"
)

const (
	// MetadataFilename is name of the file where clustermetadata is stored.
	MetadataFilename = "metadata.json"

	stateFileName = "terraform.tfstate"
)

// Cluster uses the terraform executable to launch a cluster
// with the given terraform tfvar and generated templates.
type Cluster struct {
	FileList []*asset.File
}

var _ asset.WritableAsset = (*Cluster)(nil)

// Name returns the human-friendly name of the asset.
func (c *Cluster) Name() string {
	return "Cluster"
}

// Dependencies returns the direct dependency for launching
// the cluster.
func (c *Cluster) Dependencies() []asset.Asset {
	return []asset.Asset{
		&installconfig.InstallConfig{},
		&TerraformVariables{},
		&kubeconfig.Admin{},
	}
}

// Generate launches the cluster and generates the terraform state file on disk.
func (c *Cluster) Generate(parents asset.Parents) (err error) {
	installConfig := &installconfig.InstallConfig{}
	terraformVariables := &TerraformVariables{}
	adminKubeconfig := &kubeconfig.Admin{}
	parents.Get(installConfig, terraformVariables, adminKubeconfig)

	// Copy the terraform.tfvars to a temp directory where the terraform will be invoked within.
	tmpDir, err := ioutil.TempDir(os.TempDir(), "openshift-install-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir for terraform execution")
	}
	defer os.RemoveAll(tmpDir)

	terraformVariablesFile := terraformVariables.Files()[0]
	if err := ioutil.WriteFile(filepath.Join(tmpDir, terraformVariablesFile.Filename), terraformVariablesFile.Data, 0600); err != nil {
		return errors.Wrap(err, "failed to write terraform.tfvars file")
	}

	platform := installConfig.Config.Platform.Name()
	if err := data.Unpack(tmpDir, platform); err != nil {
		return err
	}

	metadata := &types.ClusterMetadata{
		ClusterName: installConfig.Config.ObjectMeta.Name,
	}

	defer func() {
		if data, err2 := json.Marshal(metadata); err2 == nil {
			c.FileList = append(c.FileList, &asset.File{
				Filename: MetadataFilename,
				Data:     data,
			})
		} else {
			err2 = errors.Wrap(err2, "failed to Marshal ClusterMetadata")
			if err == nil {
				err = err2
			} else {
				logrus.Error(err2)
			}
		}
		// serialize metadata and stuff it into c.FileList
	}()

	switch {
	case installConfig.Config.Platform.AWS != nil:
		metadata.ClusterPlatformMetadata.AWS = &types.ClusterAWSPlatformMetadata{
			Region: installConfig.Config.Platform.AWS.Region,
			Identifier: map[string]string{
				"tectonicClusterID": installConfig.Config.ClusterID,
			},
		}
	case installConfig.Config.Platform.OpenStack != nil:
		metadata.ClusterPlatformMetadata.OpenStack = &types.ClusterOpenStackPlatformMetadata{
			Region: installConfig.Config.Platform.OpenStack.Region,
			Identifier: map[string]string{
				"tectonicClusterID": installConfig.Config.ClusterID,
			},
		}
	case installConfig.Config.Platform.Libvirt != nil:
		metadata.ClusterPlatformMetadata.Libvirt = &types.ClusterLibvirtPlatformMetadata{
			URI: installConfig.Config.Platform.Libvirt.URI,
		}
	default:
		return fmt.Errorf("no known platform")
	}

	if err := data.Unpack(filepath.Join(tmpDir, "config.tf"), "config.tf"); err != nil {
		return err
	}

	logrus.Infof("Using Terraform to create cluster...")

	// This runs the terraform in a temp directory, the tfstate file will be returned
	// to the asset store to persist it on the disk.
	if err := terraform.Init(tmpDir); err != nil {
		return errors.Wrap(err, "failed to initialize terraform")
	}

	stateFile, err := terraform.Apply(tmpDir)
	if err != nil {
		err = errors.Wrap(err, "failed to run terraform")
	}

	data, err2 := ioutil.ReadFile(stateFile)
	if err2 == nil {
		c.FileList = append(c.FileList, &asset.File{
			Filename: stateFileName,
			Data:     data,
		})
	} else {
		if err == nil {
			err = err2
		} else {
			logrus.Errorf("Failed to read tfstate: %v", err2)
		}
	}

	// TODO(yifan): Use the kubeconfig to verify the cluster is up.
	return err
}

// Files returns the FileList generated by the asset.
func (c *Cluster) Files() []*asset.File {
	return c.FileList
}

// Load returns error if the tfstate file is already on-disk, because we want to
// prevent user from accidentally re-launching the cluster.
func (c *Cluster) Load(f asset.FileFetcher) (found bool, err error) {
	if f.FetchByName(stateFileName) != nil {
		return true, fmt.Errorf("%q already exisits", stateFileName)
	}
	return false, nil
}
