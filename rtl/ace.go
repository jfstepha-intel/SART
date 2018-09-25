package rtl

var AceStructs = map[string]int {
    "wlcclnt_ifsbd_snpd": 0,
    "wlcclnt_ec0glcnt2ac1t02x5": 1,
    "wlcclnt_ec0glext0ac1a02x5": 2,
    "pmsrvr_mspmas": 3,
    "cha_datap_cms_cabist_satellite_0": 4,
    "cha_datap_cms_cabist_gclk_make_lcb_loc_and_48": 5,
}

func (m Module) Aceness() (bool, int) {
    if aceid, found := AceStructs[m.Name]; found {
        return true, aceid
    }
    return false, 0
}

func MaxAce() int {
    return len(AceStructs)
}
