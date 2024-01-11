package main

type LivecallsList struct {
	Uuid             string `json:"uuid"`
	Direction        string `json:"direction"`
	Created          string `json:"created"`
	CreatedEpoch     string `json:"created_epoch"`
	Name             string `json:"name"`
	State            string `json:"state"`
	CidName          string `json:"cid_name"`
	CidNum           string `json:"cid_num"`
	IpAddr           string `json:"ip_addr"`
	Dest             string `json:"dest"`
	PresenceId       string `json:"presence_id"`
	PresenceData     string `json:"presence_data"`
	Accountcode      string `json:"accountcode"`
	Callstate        string `json:"callstate"`
	CalleeName       string `json:"callee_name"`
	CalleeNum        string `json:"callee_num"`
	CalleeDirection  string `json:"callee_direction"`
	CallUuid         string `json:"call_uuid"`
	Hostname         string `json:"hostname"`
	SentCalleeName   string `json:"sent_callee_name"`
	SentCalleeNum    string `json:"sent_callee_num"`
	BUuid            string `json:"b_uuid"`
	BDirection       string `json:"b_direction"`
	BCreated         string `json:"b_created"`
	BCreatedEpoch    string `json:"b_created_epoch"`
	BName            string `json:"b_name"`
	BState           string `json:"b_state"`
	BCidName         string `json:"b_cid_name"`
	BCidNum          string `json:"b_cid_num"`
	BIpAddr          string `json:"b_ip_addr"`
	BDest            string `json:"b_dest"`
	BPresenceId      string `json:"b_presence_id"`
	BPresenceData    string `json:"b_presence_data"`
	BAccountCode     string `json:"b_accountcode"`
	BCallstate       string `json:"b_callstate"`
	BCalleeName      string `json:"b_callee_name"`
	BCalleeNum       string `json:"b_callee_num"`
	BCalleeDirection string `json:"b_callee_direction"`
	BSentCalleeName  string `json:"b_sent_callee_name"`
	BSentCalleeNum   string `json:"b_sent_callee_num"`
	CallCreatedEpoch string `json:"call_created_epoch"`
}

type Livecalls struct {
	RowCount int             `json:"row_count"`
	Rows     []LivecallsList `json:"rows"`
}
