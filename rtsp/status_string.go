// generated by stringer -type=Status status.go; DO NOT EDIT

package rtsp

import "fmt"

const _Status_name = "ContinueOkCreatedLowOnStorageMultipleChoicesMovedPermanentlySeeOtherUseProxyBadRequestUnauthorizedPaymentRequiredForbiddenNotFoundMethodNotAllowedNotAcceptableProxyAuthenticationRequiredRequestTimeoutGoneLengthRequiredPreconditionFailedRequestEntityTooLargeRequestURITooLongUnsupportedMediaTypeInvalidparameterIllegalConferenceIdentifierNotEnoughBandwidthSessionNotFoundMethodNotValidInThisStateHeaderFieldNotValidInvalidRangeParameterIsReadOnlyAggregateOperationNotAllowedOnlyAggregateOperationAllowedUnsupportedTransportDestinationUnreachableInternalServerErrorNotImplementedBadGatewayServiceUnavailableGatewayTimeoutRTSPVersionNotSupportedOptionnotsupport"

var _Status_map = map[Status]string{
	100: _Status_name[0:8],
	200: _Status_name[8:10],
	201: _Status_name[10:17],
	250: _Status_name[17:29],
	300: _Status_name[29:44],
	301: _Status_name[44:60],
	303: _Status_name[60:68],
	305: _Status_name[68:76],
	400: _Status_name[76:86],
	401: _Status_name[86:98],
	402: _Status_name[98:113],
	403: _Status_name[113:122],
	404: _Status_name[122:130],
	405: _Status_name[130:146],
	406: _Status_name[146:159],
	407: _Status_name[159:186],
	408: _Status_name[186:200],
	410: _Status_name[200:204],
	411: _Status_name[204:218],
	412: _Status_name[218:236],
	413: _Status_name[236:257],
	414: _Status_name[257:274],
	415: _Status_name[274:294],
	451: _Status_name[294:310],
	452: _Status_name[310:337],
	453: _Status_name[337:355],
	454: _Status_name[355:370],
	455: _Status_name[370:395],
	456: _Status_name[395:414],
	457: _Status_name[414:426],
	458: _Status_name[426:445],
	459: _Status_name[445:473],
	460: _Status_name[473:502],
	461: _Status_name[502:522],
	462: _Status_name[522:544],
	500: _Status_name[544:563],
	501: _Status_name[563:577],
	502: _Status_name[577:587],
	503: _Status_name[587:605],
	504: _Status_name[605:619],
	505: _Status_name[619:642],
	551: _Status_name[642:658],
}

func (i Status) String() string {
	if str, ok := _Status_map[i]; ok {
		return str
	}
	return fmt.Sprintf("Status(%d)", i)
}
