resource "pangolin_label_attachment" "site" {
  org_id   = pangolin_organization.example.org_id
  label_id = pangolin_label.production.label_id
  site_id  = pangolin_site.example.site_id
}

resource "pangolin_label_attachment" "resource" {
  org_id      = pangolin_organization.example.org_id
  label_id    = pangolin_label.production.label_id
  resource_id = pangolin_resource.example.resource_id
}
