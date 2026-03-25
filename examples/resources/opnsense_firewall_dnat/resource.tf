resource "opnsense_firewall_dnat" "example_port_forward" {
  interface = "wan"
  protocol  = "tcp"

  destination = {
    net  = "wanip"
    port = "8080"
  }

  target = {
    ip   = "192.168.1.100"
    port = "80"
  }

  log         = true
  description = "Forward WAN:8080 to internal web server"
}

resource "opnsense_firewall_dnat" "example_with_auto_rule" {
  interface = "wan"
  protocol  = "tcp"

  destination = {
    net  = "wanip"
    port = "443"
  }

  target = {
    ip   = "192.168.1.100"
    port = "443"
  }

  pass        = "pass"
  description = "Forward HTTPS with auto firewall rule"
}

resource "opnsense_firewall_dnat" "example_udp" {
  interface = "wan"
  protocol  = "udp"

  destination = {
    net  = "wanip"
    port = "51820"
  }

  target = {
    ip   = "192.168.1.50"
    port = "51820"
  }

  nat_reflection = "enable"
  description    = "Forward WireGuard UDP"
}
