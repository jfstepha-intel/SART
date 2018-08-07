package rtl

var AceStructs = map[string]uint64 {
    "wlcclnt_ifsbd_snpd": 1,
    "wlcclnt_ec0glcnt2ac1t02x5": 2,
    "wlcclnt_ec0glext0ac1a02x5": 3,
}

func (m Module) Aceness() (bool, uint64) {
    if aceid, found := AceStructs[m.Name]; found {
        return true, aceid
    }
    return false, 0
}
