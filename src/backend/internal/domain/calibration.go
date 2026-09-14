package domain

type ThresholdCalibrationPoint struct {
	Threshold float64
	Precision float64
	Recall    float64
	F1        float64
}

var CalibrationPoints = []ThresholdCalibrationPoint{
	{Threshold: 0.05, Precision: 0.0872, Recall: 0.9966, F1: 0.1604},
	{Threshold: 0.06, Precision: 0.0875, Recall: 0.9958, F1: 0.1608},
	{Threshold: 0.07, Precision: 0.0878, Recall: 0.9942, F1: 0.1614},
	{Threshold: 0.08, Precision: 0.0882, Recall: 0.9908, F1: 0.1620},
	{Threshold: 0.09, Precision: 0.0886, Recall: 0.9876, F1: 0.1626},
	{Threshold: 0.10, Precision: 0.0894, Recall: 0.9863, F1: 0.1640},
	{Threshold: 0.11, Precision: 0.0903, Recall: 0.9858, F1: 0.1655},
	{Threshold: 0.12, Precision: 0.0913, Recall: 0.9852, F1: 0.1672},
	{Threshold: 0.13, Precision: 0.0922, Recall: 0.9836, F1: 0.1687},
	{Threshold: 0.14, Precision: 0.0931, Recall: 0.9811, F1: 0.1701},
	{Threshold: 0.15, Precision: 0.0941, Recall: 0.9781, F1: 0.1716},
	{Threshold: 0.16, Precision: 0.0949, Recall: 0.9745, F1: 0.1729},
	{Threshold: 0.17, Precision: 0.0958, Recall: 0.9721, F1: 0.1744},
	{Threshold: 0.18, Precision: 0.0967, Recall: 0.9690, F1: 0.1758},
	{Threshold: 0.19, Precision: 0.0977, Recall: 0.9671, F1: 0.1775},
	{Threshold: 0.20, Precision: 0.0986, Recall: 0.9634, F1: 0.1789},
	{Threshold: 0.21, Precision: 0.0993, Recall: 0.9566, F1: 0.1800},
	{Threshold: 0.22, Precision: 0.1004, Recall: 0.9536, F1: 0.1816},
	{Threshold: 0.23, Precision: 0.1014, Recall: 0.9490, F1: 0.1832},
	{Threshold: 0.24, Precision: 0.1025, Recall: 0.9442, F1: 0.1850},
	{Threshold: 0.25, Precision: 0.1039, Recall: 0.9399, F1: 0.1871},
	{Threshold: 0.26, Precision: 0.1050, Recall: 0.9323, F1: 0.1887},
	{Threshold: 0.27, Precision: 0.1061, Recall: 0.9247, F1: 0.1904},
	{Threshold: 0.28, Precision: 0.1072, Recall: 0.9174, F1: 0.1920},
	{Threshold: 0.29, Precision: 0.1086, Recall: 0.9112, F1: 0.1941},
	{Threshold: 0.30, Precision: 0.1100, Recall: 0.9055, F1: 0.1962},
	{Threshold: 0.31, Precision: 0.1114, Recall: 0.8975, F1: 0.1982},
	{Threshold: 0.32, Precision: 0.1127, Recall: 0.8866, F1: 0.1999},
	{Threshold: 0.33, Precision: 0.1140, Recall: 0.8758, F1: 0.2018},
	{Threshold: 0.34, Precision: 0.1154, Recall: 0.8629, F1: 0.2036},
	{Threshold: 0.35, Precision: 0.1169, Recall: 0.8502, F1: 0.2055},
	{Threshold: 0.36, Precision: 0.1189, Recall: 0.8402, F1: 0.2084},
	{Threshold: 0.37, Precision: 0.1206, Recall: 0.8259, F1: 0.2105},
	{Threshold: 0.38, Precision: 0.1225, Recall: 0.8125, F1: 0.2129},
	{Threshold: 0.39, Precision: 0.1242, Recall: 0.7970, F1: 0.2149},
	{Threshold: 0.40, Precision: 0.1263, Recall: 0.7796, F1: 0.2174},
	{Threshold: 0.41, Precision: 0.1285, Recall: 0.7620, F1: 0.2199},
	{Threshold: 0.42, Precision: 0.1308, Recall: 0.7443, F1: 0.2225},
	{Threshold: 0.43, Precision: 0.1336, Recall: 0.7265, F1: 0.2257},
	{Threshold: 0.44, Precision: 0.1363, Recall: 0.7025, F1: 0.2282},
	{Threshold: 0.45, Precision: 0.1385, Recall: 0.6740, F1: 0.2297},
	{Threshold: 0.46, Precision: 0.1418, Recall: 0.6503, F1: 0.2328},
	{Threshold: 0.47, Precision: 0.1463, Recall: 0.6261, F1: 0.2372},
	{Threshold: 0.48, Precision: 0.1512, Recall: 0.6005, F1: 0.2416},
	{Threshold: 0.49, Precision: 0.1554, Recall: 0.5680, F1: 0.2440},
	{Threshold: 0.50, Precision: 0.1598, Recall: 0.5348, F1: 0.2461},
	{Threshold: 0.51, Precision: 0.1663, Recall: 0.5052, F1: 0.2502},
	{Threshold: 0.52, Precision: 0.1741, Recall: 0.4765, F1: 0.2550},
	{Threshold: 0.53, Precision: 0.1794, Recall: 0.4397, F1: 0.2548},
	{Threshold: 0.54, Precision: 0.1855, Recall: 0.4045, F1: 0.2544},
	{Threshold: 0.55, Precision: 0.1916, Recall: 0.3700, F1: 0.2525},
	{Threshold: 0.56, Precision: 0.1965, Recall: 0.3342, F1: 0.2475},
	{Threshold: 0.57, Precision: 0.2010, Recall: 0.2978, F1: 0.2400},
	{Threshold: 0.58, Precision: 0.2073, Recall: 0.2659, F1: 0.2330},
	{Threshold: 0.59, Precision: 0.2149, Recall: 0.2390, F1: 0.2263},
	{Threshold: 0.60, Precision: 0.2212, Recall: 0.2135, F1: 0.2173},
	{Threshold: 0.61, Precision: 0.2290, Recall: 0.1895, F1: 0.2074},
	{Threshold: 0.62, Precision: 0.2431, Recall: 0.1730, F1: 0.2022},
	{Threshold: 0.63, Precision: 0.2518, Recall: 0.1506, F1: 0.1885},
	{Threshold: 0.64, Precision: 0.2612, Recall: 0.1340, F1: 0.1771},
	{Threshold: 0.65, Precision: 0.2758, Recall: 0.1222, F1: 0.1694},
	{Threshold: 0.66, Precision: 0.2849, Recall: 0.1066, F1: 0.1551},
	{Threshold: 0.67, Precision: 0.3022, Recall: 0.0945, F1: 0.1440},
	{Threshold: 0.68, Precision: 0.3051, Recall: 0.0806, F1: 0.1275},
	{Threshold: 0.69, Precision: 0.3169, Recall: 0.0703, F1: 0.1151},
	{Threshold: 0.70, Precision: 0.3241, Recall: 0.0601, F1: 0.1015},
	{Threshold: 0.71, Precision: 0.3433, Recall: 0.0518, F1: 0.0900},
	{Threshold: 0.72, Precision: 0.3604, Recall: 0.0443, F1: 0.0790},
	{Threshold: 0.73, Precision: 0.3785, Recall: 0.0369, F1: 0.0673},
	{Threshold: 0.74, Precision: 0.4004, Recall: 0.0308, F1: 0.0572},
	{Threshold: 0.75, Precision: 0.4058, Recall: 0.0247, F1: 0.0465},
	{Threshold: 0.76, Precision: 0.4040, Recall: 0.0193, F1: 0.0369},
	{Threshold: 0.77, Precision: 0.4069, Recall: 0.0152, F1: 0.0292},
	{Threshold: 0.78, Precision: 0.4138, Recall: 0.0116, F1: 0.0226},
	{Threshold: 0.79, Precision: 0.3893, Recall: 0.0082, F1: 0.0161},
	{Threshold: 0.80, Precision: 0.4043, Recall: 0.0061, F1: 0.0121},
	{Threshold: 0.81, Precision: 0.3768, Recall: 0.0042, F1: 0.0083},
	{Threshold: 0.82, Precision: 0.3509, Recall: 0.0032, F1: 0.0064},
	{Threshold: 0.83, Precision: 0.3095, Recall: 0.0021, F1: 0.0042},
	{Threshold: 0.84, Precision: 0.3200, Recall: 0.0013, F1: 0.0026},
	{Threshold: 0.85, Precision: 0.1818, Recall: 0.0003, F1: 0.0006},
	{Threshold: 0.86, Precision: 0.2500, Recall: 0.0003, F1: 0.0006},
	{Threshold: 0.87, Precision: 0.2000, Recall: 0.0002, F1: 0.0003},
	{Threshold: 0.88, Precision: 0.0000, Recall: 0.0000, F1: 0.0000},
	{Threshold: 0.89, Precision: 0.0000, Recall: 0.0000, F1: 0.0000},
	{Threshold: 0.90, Precision: 0.0000, Recall: 0.0000, F1: 0.0000},
	{Threshold: 0.91, Precision: 0.0000, Recall: 0.0000, F1: 0.0000},
	{Threshold: 0.92, Precision: 0.0000, Recall: 0.0000, F1: 0.0000},
	{Threshold: 0.93, Precision: 0.0000, Recall: 0.0000, F1: 0.0000},
	{Threshold: 0.94, Precision: 0.0000, Recall: 0.0000, F1: 0.0000},
	{Threshold: 0.95, Precision: 0.0000, Recall: 0.0000, F1: 0.0000},
}

func GetThresholdCalibrationMetrics(threshold float64) (precision, recall, f1 float64) {
	if len(CalibrationPoints) == 0 {
		return 0.159, 0.553, 0.247
	}
	if threshold <= CalibrationPoints[0].Threshold {
		p := CalibrationPoints[0]
		return p.Precision, p.Recall, p.F1
	}
	if threshold >= CalibrationPoints[len(CalibrationPoints)-1].Threshold {
		p := CalibrationPoints[len(CalibrationPoints)-1]
		return p.Precision, p.Recall, p.F1
	}
	for i := 0; i < len(CalibrationPoints)-1; i++ {
		p1 := CalibrationPoints[i]
		p2 := CalibrationPoints[i+1]
		if threshold >= p1.Threshold && threshold <= p2.Threshold {
			factor := (threshold - p1.Threshold) / (p2.Threshold - p1.Threshold)
			prec := p1.Precision + factor*(p2.Precision-p1.Precision)
			rec := p1.Recall + factor*(p2.Recall-p1.Recall)
			f1Val := p1.F1 + factor*(p2.F1-p1.F1)
			return prec, rec, f1Val
		}
	}
	return 0.159, 0.553, 0.247
}
