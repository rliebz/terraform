package google

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/api/compute/v1"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccInstanceGroupManager_basic(t *testing.T) {
	var manager compute.InstanceGroupManager

	template := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	target := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm1 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm2 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManager_basic(template, target, igm1, igm2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-basic", &manager),
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-no-tp", &manager),
				),
			},
		},
	})
}

func TestAccInstanceGroupManager_update(t *testing.T) {
	var manager compute.InstanceGroupManager

	template1 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	target := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	template2 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManager_update(template1, target, igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-update", &manager),
					testAccCheckInstanceGroupManagerNamedPorts(
						"google_compute_instance_group_manager.igm-update",
						map[string]int64{"customhttp": 8080},
						&manager),
				),
			},
			resource.TestStep{
				Config: testAccInstanceGroupManager_update2(template1, target, template2, igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-update", &manager),
					testAccCheckInstanceGroupManagerUpdated(
						"google_compute_instance_group_manager.igm-update", 3,
						"google_compute_target_pool.igm-update", template2),
					testAccCheckInstanceGroupManagerNamedPorts(
						"google_compute_instance_group_manager.igm-update",
						map[string]int64{"customhttp": 8080, "customhttps": 8443},
						&manager),
				),
			},
		},
	})
}

func TestAccInstanceGroupManager_updateLifecycle(t *testing.T) {
	var manager compute.InstanceGroupManager

	tag1 := "tag1"
	tag2 := "tag2"
	igm := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManager_updateLifecycle(tag1, igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-update", &manager),
				),
			},
			resource.TestStep{
				Config: testAccInstanceGroupManager_updateLifecycle(tag2, igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-update", &manager),
					testAccCheckInstanceGroupManagerTemplateTags(
						"google_compute_instance_group_manager.igm-update", []string{tag2}),
				),
			},
		},
	})
}

func TestAccInstanceGroupManager_updateStrategy(t *testing.T) {
	var manager compute.InstanceGroupManager
	igm := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManager_updateStrategy(igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-update-strategy", &manager),
					testAccCheckInstanceGroupManagerUpdateStrategy(
						"google_compute_instance_group_manager.igm-update-strategy", "NONE"),
				),
			},
		},
	})
}

func TestAccInstanceGroupManager_separateRegions(t *testing.T) {
	var manager compute.InstanceGroupManager

	igm1 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm2 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManager_separateRegions(igm1, igm2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-basic", &manager),
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-basic-2", &manager),
				),
			},
		},
	})
}

func testAccCheckInstanceGroupManagerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_instance_group_manager" {
			continue
		}
		_, err := config.clientCompute.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("InstanceGroupManager still exists")
		}
	}

	return nil
}

func testAccCheckInstanceGroupManagerExists(n string, manager *compute.InstanceGroupManager) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("InstanceGroupManager not found")
		}

		*manager = *found

		return nil
	}
}

func testAccCheckInstanceGroupManagerUpdated(n string, size int64, targetPool string, template string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		manager, err := config.clientCompute.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		// Cannot check the target pool as the instance creation is asynchronous.  However, can
		// check the target_size.
		if manager.TargetSize != size {
			return fmt.Errorf("instance count incorrect")
		}

		// check that the instance template updated
		instanceTemplate, err := config.clientCompute.InstanceTemplates.Get(
			config.Project, template).Do()
		if err != nil {
			return fmt.Errorf("Error reading instance template: %s", err)
		}

		if instanceTemplate.Name != template {
			return fmt.Errorf("instance template not updated")
		}

		return nil
	}
}

func testAccCheckInstanceGroupManagerNamedPorts(n string, np map[string]int64, instanceGroupManager *compute.InstanceGroupManager) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		manager, err := config.clientCompute.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		var found bool
		for _, namedPort := range manager.NamedPorts {
			found = false
			for name, port := range np {
				if namedPort.Name == name && namedPort.Port == port {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("named port incorrect")
			}
		}

		return nil
	}
}

func testAccCheckInstanceGroupManagerTemplateTags(n string, tags []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		manager, err := config.clientCompute.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		// check that the instance template updated
		instanceTemplate, err := config.clientCompute.InstanceTemplates.Get(
			config.Project, resourceSplitter(manager.InstanceTemplate)).Do()
		if err != nil {
			return fmt.Errorf("Error reading instance template: %s", err)
		}

		if !reflect.DeepEqual(instanceTemplate.Properties.Tags.Items, tags) {
			return fmt.Errorf("instance template not updated")
		}

		return nil
	}
}

func testAccCheckInstanceGroupManagerUpdateStrategy(n, strategy string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if rs.Primary.Attributes["update_strategy"] != strategy {
			return fmt.Errorf("Expected strategy to be %s, got %s",
				strategy, rs.Primary.Attributes["update_strategy"])
		}
		return nil
	}
}

func testAccInstanceGroupManager_basic(template, target, igm1, igm2 string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-basic" {
		name = "%s"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}
	}

	resource "google_compute_target_pool" "igm-basic" {
		description = "Resource created for Terraform acceptance testing"
		name = "%s"
		session_affinity = "CLIENT_IP_PROTO"
	}

	resource "google_compute_instance_group_manager" "igm-basic" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-basic.self_link}"
		target_pools = ["${google_compute_target_pool.igm-basic.self_link}"]
		base_instance_name = "igm-basic"
		zone = "us-central1-c"
		target_size = 2
	}

	resource "google_compute_instance_group_manager" "igm-no-tp" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-basic.self_link}"
		base_instance_name = "igm-no-tp"
		zone = "us-central1-c"
		target_size = 2
	}
	`, template, target, igm1, igm2)
}

func testAccInstanceGroupManager_update(template, target, igm string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-update" {
		name = "%s"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}
	}

	resource "google_compute_target_pool" "igm-update" {
		description = "Resource created for Terraform acceptance testing"
		name = "%s"
		session_affinity = "CLIENT_IP_PROTO"
	}

	resource "google_compute_instance_group_manager" "igm-update" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-update.self_link}"
		target_pools = ["${google_compute_target_pool.igm-update.self_link}"]
		base_instance_name = "igm-update"
		zone = "us-central1-c"
		target_size = 2
		named_port {
			name = "customhttp"
			port = 8080
		}
	}`, template, target, igm)
}

// Change IGM's instance template and target size
func testAccInstanceGroupManager_update2(template1, target, template2, igm string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-update" {
		name = "%s"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}
	}

	resource "google_compute_target_pool" "igm-update" {
		description = "Resource created for Terraform acceptance testing"
		name = "%s"
		session_affinity = "CLIENT_IP_PROTO"
	}

	resource "google_compute_instance_template" "igm-update2" {
		name = "%s"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}
	}

	resource "google_compute_instance_group_manager" "igm-update" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-update2.self_link}"
		target_pools = ["${google_compute_target_pool.igm-update.self_link}"]
		base_instance_name = "igm-update"
		zone = "us-central1-c"
		target_size = 3
		named_port {
			name = "customhttp"
			port = 8080
		}
		named_port {
			name = "customhttps"
			port = 8443
		}
	}`, template1, target, template2, igm)
}

func testAccInstanceGroupManager_updateLifecycle(tag, igm string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-update" {
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["%s"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}

		lifecycle {
			create_before_destroy = true
		}
	}

	resource "google_compute_instance_group_manager" "igm-update" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-update.self_link}"
		base_instance_name = "igm-update"
		zone = "us-central1-c"
		target_size = 2
		named_port {
			name = "customhttp"
			port = 8080
		}
	}`, tag, igm)
}

func testAccInstanceGroupManager_updateStrategy(igm string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-update-strategy" {
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["terraform-testing"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}

		lifecycle {
			create_before_destroy = true
		}
	}

	resource "google_compute_instance_group_manager" "igm-update-strategy" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-update-strategy.self_link}"
		base_instance_name = "igm-update-strategy"
		zone = "us-central1-c"
		target_size = 2
		update_strategy = "NONE"
		named_port {
			name = "customhttp"
			port = 8080
		}
	}`, igm)
}

func testAccInstanceGroupManager_separateRegions(igm1, igm2 string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-basic" {
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}
	}

	resource "google_compute_instance_group_manager" "igm-basic" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-basic.self_link}"
		base_instance_name = "igm-basic"
		zone = "us-central1-c"
		target_size = 2
	}

	resource "google_compute_instance_group_manager" "igm-basic-2" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-basic.self_link}"
		base_instance_name = "igm-basic-2"
		zone = "us-west1-b"
		target_size = 2
	}
	`, igm1, igm2)
}

func resourceSplitter(resource string) string {
	splits := strings.Split(resource, "/")

	return splits[len(splits)-1]
}
