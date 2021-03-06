package cidranger

import (
	"net"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsert(t *testing.T) {
	ranger := newBruteRanger().(*bruteRanger)
	_, networkIPv4, _ := net.ParseCIDR("0.0.1.0/24")
	_, networkIPv6, _ := net.ParseCIDR("8000::/96")
	entryIPv4 := NewBasicRangerEntry(*networkIPv4)
	entryIPv6 := NewBasicRangerEntry(*networkIPv6)

	ranger.Insert(entryIPv4)
	ranger.Insert(entryIPv6)

	assert.Equal(t, 1, len(ranger.ipV4Entries))
	assert.Equal(t, entryIPv4, ranger.ipV4Entries["0.0.1.0/24"])
	assert.Equal(t, 1, len(ranger.ipV6Entries))
	assert.Equal(t, entryIPv6, ranger.ipV6Entries["8000::/96"])
}

func TestInsertError(t *testing.T) {
	bRanger := newBruteRanger().(*bruteRanger)
	_, networkIPv4, _ := net.ParseCIDR("0.0.1.0/24")
	networkIPv4.IP = append(networkIPv4.IP, byte(4))
	err := bRanger.Insert(NewBasicRangerEntry(*networkIPv4))
	assert.Equal(t, ErrInvalidNetworkInput, err)
}

func TestRemove(t *testing.T) {
	ranger := newBruteRanger().(*bruteRanger)
	_, networkIPv4, _ := net.ParseCIDR("0.0.1.0/24")
	_, networkIPv6, _ := net.ParseCIDR("8000::/96")
	_, notInserted, _ := net.ParseCIDR("8000::/96")

	insertIPv4 := NewBasicRangerEntry(*networkIPv4)
	insertIPv6 := NewBasicRangerEntry(*networkIPv6)

	ranger.Insert(insertIPv4)
	deletedIPv4, err := ranger.Remove(*networkIPv4)
	assert.NoError(t, err)

	ranger.Insert(insertIPv6)
	deletedIPv6, err := ranger.Remove(*networkIPv6)
	assert.NoError(t, err)

	entry, err := ranger.Remove(*notInserted)
	assert.NoError(t, err)
	assert.Nil(t, entry)

	assert.Equal(t, insertIPv4, deletedIPv4)
	assert.Equal(t, 0, len(ranger.ipV4Entries))
	assert.Equal(t, insertIPv6, deletedIPv6)
	assert.Equal(t, 0, len(ranger.ipV6Entries))
}

func TestRemoveError(t *testing.T) {
	r := newBruteRanger().(*bruteRanger)
	_, invalidNetwork, _ := net.ParseCIDR("0.0.1.0/24")
	invalidNetwork.IP = append(invalidNetwork.IP, byte(4))

	_, err := r.Remove(*invalidNetwork)
	assert.Equal(t, ErrInvalidNetworkInput, err)
}

func TestGet(t *testing.T) {
	ranger := newBruteRanger().(*bruteRanger)
	_, networkIPv4, _ := net.ParseCIDR("0.0.1.0/24")
	_, networkIPv6, _ := net.ParseCIDR("8000::/96")
	_, notInserted, _ := net.ParseCIDR("8000::/128")

	insertIPv4 := NewBasicRangerEntry(*networkIPv4)
	insertIPv6 := NewBasicRangerEntry(*networkIPv6)

	ranger.Insert(insertIPv4)
	getIPv4, err := ranger.Get(*networkIPv4)
	assert.NoError(t, err)

	ranger.Insert(insertIPv6)
	getIPv6, err := ranger.Get(*networkIPv6)
	assert.NoError(t, err)

	entry, err := ranger.Get(*notInserted)
	assert.Error(t, ErrNoExactMatch, err)
	assert.Nil(t, entry)

	assert.Equal(t, insertIPv4, getIPv4)
	assert.Equal(t, 1, len(ranger.ipV4Entries))
	assert.Equal(t, insertIPv6, getIPv6)
	assert.Equal(t, 1, len(ranger.ipV6Entries))
}

func TestContainsExact(t *testing.T) {
	ranger := newBruteRanger().(*bruteRanger)
	_, networkIPv4, _ := net.ParseCIDR("0.0.1.0/24")
	_, networkIPv6, _ := net.ParseCIDR("8000::/96")
	_, notInserted, _ := net.ParseCIDR("8000::/128")

	insertIPv4 := NewBasicRangerEntry(*networkIPv4)
	insertIPv6 := NewBasicRangerEntry(*networkIPv6)

	ranger.Insert(insertIPv4)
	getIPv4, err := ranger.ContainsExact(*networkIPv4)
	assert.NoError(t, err)
	assert.True(t, getIPv4)

	ranger.Insert(insertIPv6)
	getIPv6, err := ranger.ContainsExact(*networkIPv6)
	assert.NoError(t, err)
	assert.True(t, getIPv6)

	entry, err := ranger.ContainsExact(*notInserted)
	assert.NoError(t, err)
	assert.False(t, entry)
}

func TestContains(t *testing.T) {
	r := newBruteRanger().(*bruteRanger)
	_, network, _ := net.ParseCIDR("0.0.1.0/24")
	_, network1, _ := net.ParseCIDR("8000::/112")
	r.Insert(NewBasicRangerEntry(*network))
	r.Insert(NewBasicRangerEntry(*network1))

	cases := []struct {
		ip       net.IP
		contains bool
		err      error
		name     string
	}{
		{net.ParseIP("0.0.1.255"), true, nil, "IPv4 should contain"},
		{net.ParseIP("0.0.0.255"), false, nil, "IPv4 shouldn't contain"},
		{net.ParseIP("8000::ffff"), true, nil, "IPv6 shouldn't contain"},
		{net.ParseIP("8000::1:ffff"), false, nil, "IPv6 shouldn't contain"},
		{append(net.ParseIP("8000::1:ffff"), byte(0)), false, ErrInvalidNetworkInput, "Invalid IP"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contains, err := r.Contains(tc.ip)
			if tc.err != nil {
				assert.Equal(t, tc.err, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.contains, contains)
			}
		})
	}
}

func TestContainingNetworks(t *testing.T) {
	r := newBruteRanger().(*bruteRanger)
	_, network1, _ := net.ParseCIDR("0.0.1.0/24")
	_, network2, _ := net.ParseCIDR("0.0.1.0/25")
	_, network3, _ := net.ParseCIDR("8000::/112")
	_, network4, _ := net.ParseCIDR("8000::/113")
	entry1 := NewBasicRangerEntry(*network1)
	entry2 := NewBasicRangerEntry(*network2)
	entry3 := NewBasicRangerEntry(*network3)
	entry4 := NewBasicRangerEntry(*network4)
	r.Insert(entry1)
	r.Insert(entry2)
	r.Insert(entry3)
	r.Insert(entry4)
	cases := []struct {
		ip                 net.IP
		containingNetworks []RangerEntry
		err                error
		name               string
	}{
		{net.ParseIP("0.0.1.255"), []RangerEntry{entry1}, nil, "IPv4 should contain"},
		{net.ParseIP("0.0.1.127"), []RangerEntry{entry1, entry2}, nil, "IPv4 should contain both"},
		{net.ParseIP("0.0.0.127"), []RangerEntry{}, nil, "IPv4 should contain none"},
		{net.ParseIP("8000::ffff"), []RangerEntry{entry3}, nil, "IPv6 should contain"},
		{net.ParseIP("8000::7fff"), []RangerEntry{entry3, entry4}, nil, "IPv6 should contain both"},
		{net.ParseIP("8000::1:7fff"), []RangerEntry{}, nil, "IPv6 should contain none"},
		{append(net.ParseIP("8000::1:7fff"), byte(0)), nil, ErrInvalidNetworkInput, "Invalid IP"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			networks, err := r.ContainingNetworks(tc.ip)
			if tc.err != nil {
				assert.Equal(t, tc.err, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tc.containingNetworks), len(networks))
				for _, network := range tc.containingNetworks {
					assert.Contains(t, networks, network)
				}
			}
		})
	}
}

func TestCoveredNetworks(t *testing.T) {
	for _, tc := range coveredNetworkTests {
		t.Run(tc.name, func(t *testing.T) {
			ranger := newBruteRanger()
			for _, insert := range tc.inserts {
				_, network, _ := net.ParseCIDR(insert)
				err := ranger.Insert(NewBasicRangerEntry(*network))
				assert.NoError(t, err)
			}
			var expectedEntries []string
			for _, network := range tc.networks {
				expectedEntries = append(expectedEntries, network)
			}
			sort.Strings(expectedEntries)
			_, snet, _ := net.ParseCIDR(tc.search)
			networks, err := ranger.CoveredNetworks(*snet)
			assert.NoError(t, err)

			var results []string
			for _, result := range networks {
				net := result.Network()
				results = append(results, net.String())
			}
			sort.Strings(results)

			assert.Equal(t, expectedEntries, results)
		})
	}
}

func TestMatchingNetworks(t *testing.T) {
	ranger := newBruteRanger().(*bruteRanger)
	_, networkSuper, _ := net.ParseCIDR("192.168.0.0/24")
	_, networkSub1, _ := net.ParseCIDR("192.168.0.0/25")
	_, networkSub2, _ := net.ParseCIDR("192.168.0.128/25")
	_, networkNot, _ := net.ParseCIDR("192.168.1.0/24")
	_, networkBad, _ := net.ParseCIDR("192.168.2.0/24")

	insertSuper := NewBasicRangerEntry(*networkSuper)
	insertSub1 := NewBasicRangerEntry(*networkSub1)
	insertSub2 := NewBasicRangerEntry(*networkSub2)
	insertNot := NewBasicRangerEntry(*networkNot)

	ranger.Insert(insertSuper)
	ranger.Insert(insertSub1)
	ranger.Insert(insertSub2)
	ranger.Insert(insertNot)

	resultsSuper := []RangerEntry{insertSuper, insertSub1, insertSub2}
	matchSuper, err := ranger.MatchingNetworks(*networkSuper)
	assert.NoError(t, err)
	assert.Equal(t, sortEntries(resultsSuper), sortEntries(matchSuper))

	resultsSub1 := []RangerEntry{insertSuper, insertSub1}
	matchSub1, err := ranger.MatchingNetworks(*networkSub1)
	assert.NoError(t, err)
	assert.Equal(t, sortEntries(resultsSub1), sortEntries(matchSub1))

	resultsSub2 := []RangerEntry{insertSuper, insertSub2}
	matchSub2, err := ranger.MatchingNetworks(*networkSub2)
	assert.NoError(t, err)
	assert.Equal(t, sortEntries(resultsSub2), sortEntries(matchSub2))

	resultsBad := []RangerEntry{}
	matchBad, err := ranger.MatchingNetworks(*networkBad)
	assert.NoError(t, err)
	assert.Equal(t, sortEntries(resultsBad), sortEntries(matchBad))
}

func sortEntries(entries []RangerEntry) []RangerEntry {
	sort.Slice(entries, func(i, j int) bool {
		iNet := entries[i].Network()
		jNet := entries[j].Network()
		if iNet.String() < jNet.String() {
			return true
		}
		return false
	})
	return entries
}
