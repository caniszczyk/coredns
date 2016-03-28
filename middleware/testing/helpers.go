package testing

import (
	"testing"

	"github.com/miekg/coredns/middleware"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

type Section int

const (
	Answer Section = iota
	Ns
	Extra
)

type RRSet []dns.RR

func (p RRSet) Len() int           { return len(p) }
func (p RRSet) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p RRSet) Less(i, j int) bool { return p[i].String() < p[j].String() }

type Case struct {
	Qname  string
	Qtype  uint16
	Rcode  int
	Answer []dns.RR
	Ns     []dns.RR
	Extra  []dns.RR
}

func A(rr string) *dns.A         { r, _ := dns.NewRR(rr); return r.(*dns.A) }
func AAAA(rr string) *dns.AAAA   { r, _ := dns.NewRR(rr); return r.(*dns.AAAA) }
func CNAME(rr string) *dns.CNAME { r, _ := dns.NewRR(rr); return r.(*dns.CNAME) }
func SRV(rr string) *dns.SRV     { r, _ := dns.NewRR(rr); return r.(*dns.SRV) }
func SOA(rr string) *dns.SOA     { r, _ := dns.NewRR(rr); return r.(*dns.SOA) }
func NS(rr string) *dns.NS       { r, _ := dns.NewRR(rr); return r.(*dns.NS) }
func PTR(rr string) *dns.PTR     { r, _ := dns.NewRR(rr); return r.(*dns.PTR) }
func TXT(rr string) *dns.TXT     { r, _ := dns.NewRR(rr); return r.(*dns.TXT) }
func MX(rr string) *dns.MX       { r, _ := dns.NewRR(rr); return r.(*dns.MX) }

func CheckSection(t *testing.T, tc Case, sect Section, rr []dns.RR) bool {
	section := []dns.RR{}
	switch sect {
	case 0:
		section = tc.Answer
	case 1:
		section = tc.Ns
	case 2:
		section = tc.Extra
	}

	for i, a := range rr {
		if a.Header().Name != section[i].Header().Name {
			t.Errorf("rr %d should have a Header Name of %q, but has %q", i, section[i].Header().Name, a.Header().Name)
			return false
		}
		// 303 signals: don't care what the ttl is.
		if section[i].Header().Ttl != 303 && a.Header().Ttl != section[i].Header().Ttl {
			t.Errorf("rr %d should have a Header TTL of %d, but has %d", i, section[i].Header().Ttl, a.Header().Ttl)
			return false
		}
		if a.Header().Rrtype != section[i].Header().Rrtype {
			t.Errorf("rr %d should have a header rr type of %d, but has %d", i, section[i].Header().Rrtype, a.Header().Rrtype)
			return false
		}

		switch x := a.(type) {
		case *dns.SRV:
			if x.Priority != section[i].(*dns.SRV).Priority {
				t.Errorf("rr %d should have a Priority of %d, but has %d", i, section[i].(*dns.SRV).Priority, x.Priority)
				return false
			}
			if x.Weight != section[i].(*dns.SRV).Weight {
				t.Errorf("rr %d should have a Weight of %d, but has %d", i, section[i].(*dns.SRV).Weight, x.Weight)
				return false
			}
			if x.Port != section[i].(*dns.SRV).Port {
				t.Errorf("rr %d should have a Port of %d, but has %d", i, section[i].(*dns.SRV).Port, x.Port)
				return false
			}
			if x.Target != section[i].(*dns.SRV).Target {
				t.Errorf("rr %d should have a Target of %q, but has %q", i, section[i].(*dns.SRV).Target, x.Target)
				return false
			}
		case *dns.A:
			if x.A.String() != section[i].(*dns.A).A.String() {
				t.Errorf("rr %d should have a Address of %q, but has %q", i, section[i].(*dns.A).A.String(), x.A.String())
				return false
			}
		case *dns.AAAA:
			if x.AAAA.String() != section[i].(*dns.AAAA).AAAA.String() {
				t.Errorf("rr %d should have a Address of %q, but has %q", i, section[i].(*dns.AAAA).AAAA.String(), x.AAAA.String())
				return false
			}
		case *dns.TXT:
			for j, txt := range x.Txt {
				if txt != section[i].(*dns.TXT).Txt[j] {
					t.Errorf("rr %d should have a Txt of %q, but has %q", i, section[i].(*dns.TXT).Txt[j], txt)
					return false
				}
			}
		case *dns.SOA:
			tt := section[i].(*dns.SOA)
			if x.Ns != tt.Ns {
				t.Errorf("SOA nameserver should be %q, but is %q", x.Ns, tt.Ns)
				return false
			}
		case *dns.PTR:
			tt := section[i].(*dns.PTR)
			if x.Ptr != tt.Ptr {
				t.Errorf("PTR ptr should be %q, but is %q", x.Ptr, tt.Ptr)
				return false
			}
		case *dns.CNAME:
			tt := section[i].(*dns.CNAME)
			if x.Target != tt.Target {
				t.Errorf("CNAME target should be %q, but is %q", x.Target, tt.Target)
				return false
			}
		case *dns.MX:
			tt := section[i].(*dns.MX)
			if x.Mx != tt.Mx {
				t.Errorf("MX Mx should be %q, but is %q", x.Mx, tt.Mx)
				return false
			}
			if x.Preference != tt.Preference {
				t.Errorf("MX Preference should be %q, but is %q", x.Preference, tt.Preference)
				return false
			}
		case *dns.NS:
			tt := section[i].(*dns.NS)
			if x.Ns != tt.Ns {
				t.Errorf("NS nameserver should be %q, but is %q", x.Ns, tt.Ns)
				return false
			}
		}
	}
	return true
}

func ErrorHandler() middleware.Handler {
	return middleware.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(m)
		return dns.RcodeServerFailure, nil
	})
}