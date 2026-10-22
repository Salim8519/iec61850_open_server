<?xml version="1.0" encoding="UTF-8"?>
<!-- ScadaBench sample IED for typed commands (ADR-0104): INC, ENC, APC, BAC, BSC and ISC in
     every control model. GGIOn/ATCCn use control model n (1 direct-normal, 2 sbo-normal,
     3 direct-enhanced, 4 sbo-enhanced). Simulation sample only; not a manufacturer model. -->
<SCL xmlns="http://www.iec.ch/61850/2003/SCL" version="2007" revision="B" release="4">
  <Header id="SampleTyped" version="1" revision="0"/>
  <IED name="SampleTyped" manufacturer="ScadaBench" type="Typed command sample" configVersion="1">
    <AccessPoint name="AP1">
      <Server>
        <Authentication none="true"/>
        <LDevice inst="CTRL">
          <LN0 lnClass="LLN0" inst="" lnType="TypLLN0"/>
          <LN lnClass="GGIO" inst="1" lnType="TypGGIO">
            <DOI name="Mod"><DAI name="ctlModel"><Val>direct-with-normal-security</Val></DAI></DOI>
            <DOI name="ISCSO1"><DAI name="ctlModel"><Val>direct-with-normal-security</Val></DAI></DOI>
            <DOI name="AnOut1"><DAI name="ctlModel"><Val>direct-with-normal-security</Val></DAI></DOI>
          </LN>
          <LN lnClass="GGIO" inst="2" lnType="TypGGIO">
            <DOI name="Mod"><DAI name="ctlModel"><Val>sbo-with-normal-security</Val></DAI></DOI>
            <DOI name="ISCSO1"><DAI name="ctlModel"><Val>sbo-with-normal-security</Val></DAI></DOI>
            <DOI name="AnOut1"><DAI name="ctlModel"><Val>sbo-with-normal-security</Val></DAI></DOI>
          </LN>
          <LN lnClass="GGIO" inst="3" lnType="TypGGIO">
            <DOI name="Mod"><DAI name="ctlModel"><Val>direct-with-enhanced-security</Val></DAI></DOI>
            <DOI name="ISCSO1"><DAI name="ctlModel"><Val>direct-with-enhanced-security</Val></DAI></DOI>
            <DOI name="AnOut1"><DAI name="ctlModel"><Val>direct-with-enhanced-security</Val></DAI></DOI>
          </LN>
          <LN lnClass="GGIO" inst="4" lnType="TypGGIO">
            <DOI name="Mod"><DAI name="ctlModel"><Val>sbo-with-enhanced-security</Val></DAI></DOI>
            <DOI name="ISCSO1"><DAI name="ctlModel"><Val>sbo-with-enhanced-security</Val></DAI></DOI>
            <DOI name="AnOut1"><DAI name="ctlModel"><Val>sbo-with-enhanced-security</Val></DAI></DOI>
          </LN>
          <LN lnClass="ATCC" inst="1" lnType="TypATCC">
            <DOI name="TapChg"><DAI name="ctlModel"><Val>direct-with-normal-security</Val></DAI></DOI>
            <DOI name="TapSet"><DAI name="ctlModel"><Val>direct-with-normal-security</Val></DAI></DOI>
            <DOI name="BndSet"><DAI name="ctlModel"><Val>direct-with-normal-security</Val></DAI></DOI>
          </LN>
          <LN lnClass="ATCC" inst="2" lnType="TypATCC">
            <DOI name="TapChg"><DAI name="ctlModel"><Val>sbo-with-normal-security</Val></DAI></DOI>
            <DOI name="TapSet"><DAI name="ctlModel"><Val>sbo-with-normal-security</Val></DAI></DOI>
            <DOI name="BndSet"><DAI name="ctlModel"><Val>sbo-with-normal-security</Val></DAI></DOI>
          </LN>
          <LN lnClass="ATCC" inst="3" lnType="TypATCC">
            <DOI name="TapChg"><DAI name="ctlModel"><Val>direct-with-enhanced-security</Val></DAI></DOI>
            <DOI name="TapSet"><DAI name="ctlModel"><Val>direct-with-enhanced-security</Val></DAI></DOI>
            <DOI name="BndSet"><DAI name="ctlModel"><Val>direct-with-enhanced-security</Val></DAI></DOI>
          </LN>
          <LN lnClass="ATCC" inst="4" lnType="TypATCC">
            <DOI name="TapChg"><DAI name="ctlModel"><Val>sbo-with-enhanced-security</Val></DAI></DOI>
            <DOI name="TapSet"><DAI name="ctlModel"><Val>sbo-with-enhanced-security</Val></DAI></DOI>
            <DOI name="BndSet"><DAI name="ctlModel"><Val>sbo-with-enhanced-security</Val></DAI></DOI>
          </LN>
        </LDevice>
      </Server>
    </AccessPoint>
  </IED>
  <DataTypeTemplates>
    <LNodeType id="TypLLN0" lnClass="LLN0">
      <DO name="Beh" type="TypENS"/>
    </LNodeType>
    <LNodeType id="TypGGIO" lnClass="GGIO">
      <DO name="Mod" type="TypENC"/>
      <DO name="ISCSO1" type="TypINC"/>
      <DO name="AnOut1" type="TypAPC"/>
    </LNodeType>
    <LNodeType id="TypATCC" lnClass="ATCC">
      <DO name="TapChg" type="TypBSC"/>
      <DO name="TapSet" type="TypISC"/>
      <DO name="BndSet" type="TypBAC"/>
    </LNodeType>
    <DOType id="TypENS" cdc="ENS">
      <DA name="stVal" bType="Enum" fc="ST" type="BehModeKind" dchg="true"><Val>on</Val></DA>
      <DA name="q" bType="Quality" fc="ST" qchg="true"/>
      <DA name="t" bType="Timestamp" fc="ST"/>
    </DOType>
    <DOType id="TypENC" cdc="ENC">
      <DA name="stVal" bType="Enum" fc="ST" type="BehModeKind" dchg="true"><Val>on</Val></DA>
      <DA name="q" bType="Quality" fc="ST" qchg="true"/>
      <DA name="t" bType="Timestamp" fc="ST"/>
      <DA name="ctlModel" bType="Enum" fc="CF" type="CtlModelKind"><Val>status-only</Val></DA>
      <DA name="sboTimeout" bType="INT32U" fc="CF"><Val>3000</Val></DA>
      <DA name="SBO" bType="VisString129" fc="CO"/>
      <DA name="Oper" bType="Struct" fc="CO" type="OperENC"/>
      <DA name="SBOw" bType="Struct" fc="CO" type="OperENC"/>
      <DA name="Cancel" bType="Struct" fc="CO" type="CancelENC"/>
    </DOType>
    <DOType id="TypINC" cdc="INC">
      <DA name="stVal" bType="INT32" fc="ST" dchg="true"><Val>0</Val></DA>
      <DA name="q" bType="Quality" fc="ST" qchg="true"/>
      <DA name="t" bType="Timestamp" fc="ST"/>
      <DA name="ctlModel" bType="Enum" fc="CF" type="CtlModelKind"><Val>status-only</Val></DA>
      <DA name="sboTimeout" bType="INT32U" fc="CF"><Val>3000</Val></DA>
      <DA name="minVal" bType="INT32" fc="CF"><Val>0</Val></DA>
      <DA name="maxVal" bType="INT32" fc="CF"><Val>10</Val></DA>
      <DA name="stepSize" bType="INT32U" fc="CF"><Val>1</Val></DA>
      <DA name="SBO" bType="VisString129" fc="CO"/>
      <DA name="Oper" bType="Struct" fc="CO" type="OperINC"/>
      <DA name="SBOw" bType="Struct" fc="CO" type="OperINC"/>
      <DA name="Cancel" bType="Struct" fc="CO" type="CancelINC"/>
    </DOType>
    <DOType id="TypAPC" cdc="APC">
      <DA name="mxVal" bType="Struct" fc="MX" type="AnalogueF" dchg="true"/>
      <DA name="q" bType="Quality" fc="MX" qchg="true"/>
      <DA name="t" bType="Timestamp" fc="MX"/>
      <DA name="ctlModel" bType="Enum" fc="CF" type="CtlModelKind"><Val>status-only</Val></DA>
      <DA name="sboTimeout" bType="INT32U" fc="CF"><Val>3000</Val></DA>
      <DA name="minVal" bType="Struct" fc="CF" type="AnalogueF"/>
      <DA name="maxVal" bType="Struct" fc="CF" type="AnalogueMax"/>
      <DA name="SBO" bType="VisString129" fc="CO"/>
      <DA name="Oper" bType="Struct" fc="CO" type="OperAPC"/>
      <DA name="SBOw" bType="Struct" fc="CO" type="OperAPC"/>
      <DA name="Cancel" bType="Struct" fc="CO" type="CancelAPC"/>
    </DOType>
    <DOType id="TypBAC" cdc="BAC">
      <DA name="mxVal" bType="Struct" fc="MX" type="AnalogueF" dchg="true"/>
      <DA name="q" bType="Quality" fc="MX" qchg="true"/>
      <DA name="t" bType="Timestamp" fc="MX"/>
      <DA name="ctlModel" bType="Enum" fc="CF" type="CtlModelKind"><Val>status-only</Val></DA>
      <DA name="sboTimeout" bType="INT32U" fc="CF"><Val>3000</Val></DA>
      <DA name="minVal" bType="Struct" fc="CF" type="AnalogueF"/>
      <DA name="maxVal" bType="Struct" fc="CF" type="AnalogueMax"/>
      <DA name="stepSize" bType="Struct" fc="CF" type="AnalogueStep"/>
      <DA name="SBO" bType="VisString129" fc="CO"/>
      <DA name="Oper" bType="Struct" fc="CO" type="OperTcmd"/>
      <DA name="SBOw" bType="Struct" fc="CO" type="OperTcmd"/>
      <DA name="Cancel" bType="Struct" fc="CO" type="CancelTcmd"/>
    </DOType>
    <DOType id="TypBSC" cdc="BSC">
      <DA name="valWTr" bType="Struct" fc="ST" type="ValWithTrans" dchg="true"/>
      <DA name="q" bType="Quality" fc="ST" qchg="true"/>
      <DA name="t" bType="Timestamp" fc="ST"/>
      <DA name="ctlModel" bType="Enum" fc="CF" type="CtlModelKind"><Val>status-only</Val></DA>
      <DA name="sboTimeout" bType="INT32U" fc="CF"><Val>3000</Val></DA>
      <DA name="SBO" bType="VisString129" fc="CO"/>
      <DA name="Oper" bType="Struct" fc="CO" type="OperTcmd"/>
      <DA name="SBOw" bType="Struct" fc="CO" type="OperTcmd"/>
      <DA name="Cancel" bType="Struct" fc="CO" type="CancelTcmd"/>
    </DOType>
    <DOType id="TypISC" cdc="ISC">
      <DA name="valWTr" bType="Struct" fc="ST" type="ValWithTrans" dchg="true"/>
      <DA name="q" bType="Quality" fc="ST" qchg="true"/>
      <DA name="t" bType="Timestamp" fc="ST"/>
      <DA name="ctlModel" bType="Enum" fc="CF" type="CtlModelKind"><Val>status-only</Val></DA>
      <DA name="sboTimeout" bType="INT32U" fc="CF"><Val>3000</Val></DA>
      <DA name="minVal" bType="INT8" fc="CF"><Val>1</Val></DA>
      <DA name="maxVal" bType="INT8" fc="CF"><Val>17</Val></DA>
      <DA name="SBO" bType="VisString129" fc="CO"/>
      <DA name="Oper" bType="Struct" fc="CO" type="OperISC"/>
      <DA name="SBOw" bType="Struct" fc="CO" type="OperISC"/>
      <DA name="Cancel" bType="Struct" fc="CO" type="CancelISC"/>
    </DOType>
    <DAType id="AnalogueF"><BDA name="f" bType="FLOAT32"><Val>0</Val></BDA></DAType>
    <DAType id="AnalogueMax"><BDA name="f" bType="FLOAT32"><Val>100</Val></BDA></DAType>
    <DAType id="AnalogueStep"><BDA name="f" bType="FLOAT32"><Val>2.5</Val></BDA></DAType>
    <DAType id="ValWithTrans">
      <BDA name="posVal" bType="INT8"><Val>5</Val></BDA>
      <BDA name="transInd" bType="BOOLEAN"><Val>false</Val></BDA>
    </DAType>
    <DAType id="Originator">
      <BDA name="orCat" bType="Enum" type="OrCat"/>
      <BDA name="orIdent" bType="Octet64"/>
    </DAType>
    <DAType id="OperENC">
      <BDA name="ctlVal" bType="Enum" type="BehModeKind"/>
      <BDA name="origin" bType="Struct" type="Originator"/>
      <BDA name="ctlNum" bType="INT8U"/>
      <BDA name="T" bType="Timestamp"/>
      <BDA name="Test" bType="BOOLEAN"/>
      <BDA name="Check" bType="Check"/>
    </DAType>
    <DAType id="CancelENC">
      <BDA name="ctlVal" bType="Enum" type="BehModeKind"/>
      <BDA name="origin" bType="Struct" type="Originator"/>
      <BDA name="ctlNum" bType="INT8U"/>
      <BDA name="T" bType="Timestamp"/>
      <BDA name="Test" bType="BOOLEAN"/>
    </DAType>
    <DAType id="OperINC">
      <BDA name="ctlVal" bType="INT32"/>
      <BDA name="origin" bType="Struct" type="Originator"/>
      <BDA name="ctlNum" bType="INT8U"/>
      <BDA name="T" bType="Timestamp"/>
      <BDA name="Test" bType="BOOLEAN"/>
      <BDA name="Check" bType="Check"/>
    </DAType>
    <DAType id="CancelINC">
      <BDA name="ctlVal" bType="INT32"/>
      <BDA name="origin" bType="Struct" type="Originator"/>
      <BDA name="ctlNum" bType="INT8U"/>
      <BDA name="T" bType="Timestamp"/>
      <BDA name="Test" bType="BOOLEAN"/>
    </DAType>
    <DAType id="OperAPC">
      <BDA name="ctlVal" bType="Struct" type="AnalogueF"/>
      <BDA name="origin" bType="Struct" type="Originator"/>
      <BDA name="ctlNum" bType="INT8U"/>
      <BDA name="T" bType="Timestamp"/>
      <BDA name="Test" bType="BOOLEAN"/>
      <BDA name="Check" bType="Check"/>
    </DAType>
    <DAType id="CancelAPC">
      <BDA name="ctlVal" bType="Struct" type="AnalogueF"/>
      <BDA name="origin" bType="Struct" type="Originator"/>
      <BDA name="ctlNum" bType="INT8U"/>
      <BDA name="T" bType="Timestamp"/>
      <BDA name="Test" bType="BOOLEAN"/>
    </DAType>
    <DAType id="OperTcmd">
      <BDA name="ctlVal" bType="Tcmd"/>
      <BDA name="origin" bType="Struct" type="Originator"/>
      <BDA name="ctlNum" bType="INT8U"/>
      <BDA name="T" bType="Timestamp"/>
      <BDA name="Test" bType="BOOLEAN"/>
      <BDA name="Check" bType="Check"/>
    </DAType>
    <DAType id="CancelTcmd">
      <BDA name="ctlVal" bType="Tcmd"/>
      <BDA name="origin" bType="Struct" type="Originator"/>
      <BDA name="ctlNum" bType="INT8U"/>
      <BDA name="T" bType="Timestamp"/>
      <BDA name="Test" bType="BOOLEAN"/>
    </DAType>
    <DAType id="OperISC">
      <BDA name="ctlVal" bType="INT8"/>
      <BDA name="origin" bType="Struct" type="Originator"/>
      <BDA name="ctlNum" bType="INT8U"/>
      <BDA name="T" bType="Timestamp"/>
      <BDA name="Test" bType="BOOLEAN"/>
      <BDA name="Check" bType="Check"/>
    </DAType>
    <DAType id="CancelISC">
      <BDA name="ctlVal" bType="INT8"/>
      <BDA name="origin" bType="Struct" type="Originator"/>
      <BDA name="ctlNum" bType="INT8U"/>
      <BDA name="T" bType="Timestamp"/>
      <BDA name="Test" bType="BOOLEAN"/>
    </DAType>
    <EnumType id="BehModeKind">
      <EnumVal ord="1">on</EnumVal>
      <EnumVal ord="2">on-blocked</EnumVal>
      <EnumVal ord="3">test</EnumVal>
      <EnumVal ord="4">test/blocked</EnumVal>
      <EnumVal ord="5">off</EnumVal>
    </EnumType>
    <EnumType id="CtlModelKind">
      <EnumVal ord="0">status-only</EnumVal>
      <EnumVal ord="1">direct-with-normal-security</EnumVal>
      <EnumVal ord="2">sbo-with-normal-security</EnumVal>
      <EnumVal ord="3">direct-with-enhanced-security</EnumVal>
      <EnumVal ord="4">sbo-with-enhanced-security</EnumVal>
    </EnumType>
    <EnumType id="OrCat">
      <EnumVal ord="0">not-supported</EnumVal>
      <EnumVal ord="1">bay-control</EnumVal>
      <EnumVal ord="2">station-control</EnumVal>
      <EnumVal ord="3">remote-control</EnumVal>
      <EnumVal ord="4">automatic-bay</EnumVal>
      <EnumVal ord="5">automatic-station</EnumVal>
      <EnumVal ord="6">automatic-remote</EnumVal>
      <EnumVal ord="7">maintenance</EnumVal>
      <EnumVal ord="8">process</EnumVal>
    </EnumType>
  </DataTypeTemplates>
</SCL>
