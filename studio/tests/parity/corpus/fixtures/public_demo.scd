<?xml version='1.0' encoding='UTF-8'?>
<SCL xmlns="http://www.iec.ch/61850/2003/SCL" xmlns:esld="https://transpower.co.nz/SCL/SSD/SLD/v0" version="2007" revision="B" release="4">
	<Header id="ScadaBench demo station" toolID="ScadaBench demo (from the OpenSCD example)" />
	<Substation name="SS1" desc="ScadaBench demo: 2 incoming 50 kV feeds, 2 busbars, 2 transformer bays" esld:w="70" esld:h="50">
		<PowerTransformer name="PT1" type="PTR" esld:x="32" esld:y="25" esld:lx="33.5" esld:ly="25">
			<TransformerWinding name="W1" type="PTW">
				<Terminal esld:uuid="ea1ac572-6f71-49fe-aa19-d18a327331cf" name="T1" connectivityNode="SS1/VL_50kV/Bay_PT1/L3" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT1" cNodeName="L3" />
			</TransformerWinding>
			</PowerTransformer>
		<PowerTransformer name="PT2" type="PTR" esld:x="41" esld:y="25" esld:lx="42.5" esld:ly="25">
			<TransformerWinding name="W1" type="PTW">
				<Terminal esld:uuid="e9cb2c53-76ca-433e-af96-c590a0ca3f90" name="T1" connectivityNode="SS1/VL_50kV/Bay_PT2/L3" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT2" cNodeName="L3" />
			</TransformerWinding>
			</PowerTransformer>
		<VoltageLevel name="VL_50kV" nomFreq="50" numPhases="3" esld:w="68" esld:h="24" esld:x="1" esld:y="1" esld:lx="1" esld:ly="1">
			<Voltage multiplier="k" unit="V">50</Voltage>
			<Bay name="BB2" esld:w="57" esld:x="8" esld:y="15" esld:lx="5.5" esld:ly="16" esld:h="1">
				<ConnectivityNode name="L" pathName="SS1/VL_50kV/BB2/L">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section bus="true">
							<esld:Vertex esld:x="8.5" esld:y="15.5" />
							<esld:Vertex esld:x="13.5" esld:y="15.5" />
						</esld:Section>
						<esld:Section bus="true">
							<esld:Vertex esld:x="13.5" esld:y="15.5" />
							<esld:Vertex esld:x="23.5" esld:y="15.5" />
						</esld:Section>
						<esld:Section bus="true">
							<esld:Vertex esld:x="23.5" esld:y="15.5" />
							<esld:Vertex esld:x="33.5" esld:y="15.5" />
						</esld:Section>
						<esld:Section bus="true">
							<esld:Vertex esld:x="33.5" esld:y="15.5" />
							<esld:Vertex esld:x="42.5" esld:y="15.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="42.5" esld:y="18.16" esld:uuid="39812065-a0d8-403c-b747-c6a29dcc2386" />
							<esld:Vertex esld:x="42.5" esld:y="15.5" />
						</esld:Section>
						<esld:Section bus="true">
							<esld:Vertex esld:x="42.5" esld:y="15.5" />
							<esld:Vertex esld:x="64.5" esld:y="15.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="33.5" esld:y="18.16" esld:uuid="79e2a905-c383-4565-9e1b-68a49665497c" />
							<esld:Vertex esld:x="33.5" esld:y="15.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="23.5" esld:y="10.84" esld:uuid="9ff9e493-4927-4fbc-bb0c-53ede0de840a" />
							<esld:Vertex esld:x="23.5" esld:y="15.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="13.5" esld:y="10.84" esld:uuid="64043ad7-ca78-4fb6-8f4f-b7da9ee21844" />
							<esld:Vertex esld:x="13.5" esld:y="15.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="7.84" esld:y="15.5" esld:uuid="02db15aa-549f-4300-9a3b-f5d94d92b8b8" />
							<esld:Vertex esld:x="8.5" esld:y="15.5" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
			</Bay>
			<Bay name="BB1" esld:w="57" esld:x="8" esld:y="13" esld:lx="5.5" esld:ly="14" esld:h="1">
				<ConnectivityNode name="L" pathName="SS1/VL_50kV/BB1/L">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section bus="true">
							<esld:Vertex esld:x="8.5" esld:y="13.5" />
							<esld:Vertex esld:x="11.5" esld:y="13.5" />
						</esld:Section>
						<esld:Section bus="true">
							<esld:Vertex esld:x="11.5" esld:y="13.5" />
							<esld:Vertex esld:x="21.5" esld:y="13.5" />
						</esld:Section>
						<esld:Section bus="true">
							<esld:Vertex esld:x="21.5" esld:y="13.5" />
							<esld:Vertex esld:x="31.5" esld:y="13.5" />
						</esld:Section>
						<esld:Section bus="true">
							<esld:Vertex esld:x="31.5" esld:y="13.5" />
							<esld:Vertex esld:x="40.5" esld:y="13.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="40.5" esld:y="18.16" esld:uuid="c97e5175-5090-45a1-a72c-fd86fbd36c1a" />
							<esld:Vertex esld:x="40.5" esld:y="13.5" />
						</esld:Section>
						<esld:Section bus="true">
							<esld:Vertex esld:x="40.5" esld:y="13.5" />
							<esld:Vertex esld:x="64.5" esld:y="13.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="31.5" esld:y="18.16" esld:uuid="56b98e94-771a-4ccd-bf8e-033d12326780" />
							<esld:Vertex esld:x="31.5" esld:y="13.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="21.5" esld:y="10.84" esld:uuid="7d6b9ce7-3e29-4753-9bee-a7d7024a7df3" />
							<esld:Vertex esld:x="21.5" esld:y="13.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="11.5" esld:y="10.84" esld:uuid="f301703d-6ab6-422a-a49b-7528e254cf2e" />
							<esld:Vertex esld:x="11.5" esld:y="13.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="7.84" esld:y="13.5" esld:uuid="db2508d2-bf2f-47d5-aa65-dfed0290e141" />
							<esld:Vertex esld:x="8.5" esld:y="13.5" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
			</Bay>
			<Bay name="Bay_Feed1" esld:x="8" esld:y="2" esld:lx="8" esld:ly="2" esld:w="9" esld:h="10">
				<ConductingEquipment type="SAR" name="SAR1" esld:x="10" esld:y="4" esld:lx="11" esld:ly="5">
					<Terminal esld:uuid="c0d691e0-0068-4b9e-b466-5c82e3a2dc50" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed1/L4" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed1" cNodeName="L4" />
				</ConductingEquipment>
				<ConductingEquipment type="IFL" name="IFL1" esld:x="12" esld:y="2" esld:lx="13" esld:ly="3" esld:rot="2">
					<Terminal esld:uuid="33919d4b-9d8c-4c37-9aa6-a0a18fde2f83" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed1/L4" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed1" cNodeName="L4" />
				</ConductingEquipment>
				<ConductingEquipment name="DIS_F1" type="DIS" esld:x="12" esld:y="4" esld:lx="13" esld:ly="5">
					<Terminal esld:uuid="c45653bb-e5c5-4991-850f-2add4e7a1bf7" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed1/L4" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed1" cNodeName="L4" />
					<Terminal esld:uuid="b42ba980-1c8c-4e4c-b5e5-6fcca701a81d" name="T2" connectivityNode="SS1/VL_50kV/Bay_Feed1/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed1" cNodeName="L1" />
				<LNode iedName="FEED1" ldInst="LD1" prefix="" lnClass="XSWI" lnInst="4" /><LNode iedName="FEED1" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="4" /></ConductingEquipment>
				<ConductingEquipment name="CTR_F1" type="CTR" esld:x="12" esld:y="6" esld:lx="13" esld:ly="7">
					<Terminal esld:uuid="8543bc92-eef6-4241-a881-651082e493b7" name="T2" connectivityNode="SS1/VL_50kV/Bay_Feed1/L2" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed1" cNodeName="L2" />
					<Terminal esld:uuid="d122addf-7e23-4b45-995d-ba307f2b293e" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed1/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed1" cNodeName="L1" />
				</ConductingEquipment>
				<ConductingEquipment name="CBR_F1" type="CBR" esld:x="12" esld:y="8" esld:lx="13" esld:ly="9">
					<Terminal esld:uuid="13ea9663-5405-47c4-b466-078f86ec3b35" name="T2" connectivityNode="SS1/VL_50kV/Bay_Feed1/L3" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed1" cNodeName="L3" />
					<Terminal esld:uuid="b7a0e336-2544-40a2-b0de-a28317d01c03" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed1/L2" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed1" cNodeName="L2" />
				<LNode iedName="FEED1" ldInst="LD1" prefix="" lnClass="XCBR" lnInst="1" /><LNode iedName="FEED1" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="1" /></ConductingEquipment>
				<ConductingEquipment name="DIS_F1_B1" type="DIS" esld:x="11" esld:y="10" esld:lx="8.5" esld:ly="11">
					<Terminal esld:uuid="f301703d-6ab6-422a-a49b-7528e254cf2e" name="T2" connectivityNode="SS1/VL_50kV/BB1/L" substationName="SS1" voltageLevelName="VL_50kV" bayName="BB1" cNodeName="L" />
					<Terminal esld:uuid="c3b376e9-ac7e-461b-8a31-49f44b5d30e5" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed1/L3" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed1" cNodeName="L3" />
				<LNode iedName="FEED1" ldInst="LD1" prefix="" lnClass="XSWI" lnInst="2" /><LNode iedName="FEED1" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="2" /></ConductingEquipment>
				<ConductingEquipment name="DIS_F1_B2" type="DIS" esld:x="13" esld:y="10" esld:lx="14" esld:ly="11">
					<Terminal esld:uuid="64043ad7-ca78-4fb6-8f4f-b7da9ee21844" name="T2" connectivityNode="SS1/VL_50kV/BB2/L" substationName="SS1" voltageLevelName="VL_50kV" bayName="BB2" cNodeName="L" />
					<Terminal esld:uuid="5b858feb-0d07-494d-858c-d8a15061305d" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed1/L3" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed1" cNodeName="L3" />
				<LNode iedName="FEED1" ldInst="LD1" prefix="" lnClass="XSWI" lnInst="3" /><LNode iedName="FEED1" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="3" /></ConductingEquipment>
				<ConnectivityNode name="L4" pathName="SS1/VL_50kV/Bay_Feed1/L4">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="12.5" esld:y="2.84" esld:uuid="33919d4b-9d8c-4c37-9aa6-a0a18fde2f83" />
							<esld:Vertex esld:x="12.5" esld:y="3.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="10.5" esld:y="4.16" esld:uuid="c0d691e0-0068-4b9e-b466-5c82e3a2dc50" />
							<esld:Vertex esld:x="10.5" esld:y="3.5" />
							<esld:Vertex esld:x="12.5" esld:y="3.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="12.5" esld:y="3.5" />
							<esld:Vertex esld:x="12.5" esld:y="4.16" esld:uuid="c45653bb-e5c5-4991-850f-2add4e7a1bf7" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
				<ConnectivityNode name="L3" pathName="SS1/VL_50kV/Bay_Feed1/L3">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="13.5" esld:y="10.16" esld:uuid="5b858feb-0d07-494d-858c-d8a15061305d" />
							<esld:Vertex esld:x="13.5" esld:y="9" />
							<esld:Vertex esld:x="12.5" esld:y="9" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="11.5" esld:y="10.16" esld:uuid="c3b376e9-ac7e-461b-8a31-49f44b5d30e5" />
							<esld:Vertex esld:x="11.5" esld:y="9" />
							<esld:Vertex esld:x="12.5" esld:y="9" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="12.5" esld:y="9" />
							<esld:Vertex esld:x="12.5" esld:y="8.84" esld:uuid="13ea9663-5405-47c4-b466-078f86ec3b35" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
				<ConnectivityNode name="L2" pathName="SS1/VL_50kV/Bay_Feed1/L2">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="12.5" esld:y="6.84" esld:uuid="8543bc92-eef6-4241-a881-651082e493b7" />
							<esld:Vertex esld:x="12.5" esld:y="8.16" esld:uuid="b7a0e336-2544-40a2-b0de-a28317d01c03" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
				<ConnectivityNode name="L1" pathName="SS1/VL_50kV/Bay_Feed1/L1">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="12.5" esld:y="4.84" esld:uuid="b42ba980-1c8c-4e4c-b5e5-6fcca701a81d" />
							<esld:Vertex esld:x="12.5" esld:y="6.16" esld:uuid="d122addf-7e23-4b45-995d-ba307f2b293e" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
			</Bay>
			<Bay name="Bay_Feed2" esld:x="18" esld:y="2" esld:lx="18" esld:ly="2" esld:w="9" esld:h="10">
				<ConductingEquipment type="SAR" name="SAR1" esld:x="20" esld:y="4" esld:lx="21" esld:ly="5">
					<Terminal esld:uuid="032e98a9-5a43-4a96-9581-0a1cb3ac0879" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed2/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed2" cNodeName="L1" />
				</ConductingEquipment>
				<ConductingEquipment type="IFL" name="IFL2" esld:x="22" esld:y="2" esld:lx="23" esld:ly="3" esld:rot="2">
					<Terminal esld:uuid="432c5857-8310-4a79-ac2f-f9247801e31c" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed2/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed2" cNodeName="L1" />
				</ConductingEquipment>
				<ConductingEquipment name="DIS_F2" type="DIS" esld:x="22" esld:y="4" esld:lx="23" esld:ly="5">
					<Terminal esld:uuid="f0acbd4c-9cac-4a8c-a95b-ff53ff57b416" name="T2" connectivityNode="SS1/VL_50kV/Bay_Feed2/L2" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed2" cNodeName="L2" />
					<Terminal esld:uuid="708adebc-8c83-404a-a604-03e76735bba7" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed2/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed2" cNodeName="L1" />
				<LNode iedName="FEED2" ldInst="LD1" prefix="" lnClass="XSWI" lnInst="4" /><LNode iedName="FEED2" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="4" /></ConductingEquipment>
				<ConductingEquipment name="CTR_F2" type="CTR" esld:x="22" esld:y="6" esld:lx="23" esld:ly="7">
					<Terminal esld:uuid="8c75a7b0-ec30-4f8a-b68b-f50796f928a2" name="T2" connectivityNode="SS1/VL_50kV/Bay_Feed2/L3" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed2" cNodeName="L3" />
					<Terminal esld:uuid="d0727ca4-059e-488f-8109-75c135b2af43" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed2/L2" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed2" cNodeName="L2" />
				</ConductingEquipment>
				<ConductingEquipment name="CBR_F2" type="CBR" esld:x="22" esld:y="8" esld:lx="23" esld:ly="9">
					<Terminal esld:uuid="a9847d00-f82f-4cb5-ad1a-548f847f9a22" name="T2" connectivityNode="SS1/VL_50kV/Bay_Feed2/L4" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed2" cNodeName="L4" />
					<Terminal esld:uuid="aa48db86-949a-4642-8862-be3ce1c46f51" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed2/L3" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed2" cNodeName="L3" />
				<LNode iedName="FEED2" ldInst="LD1" prefix="" lnClass="XCBR" lnInst="1" /><LNode iedName="FEED2" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="1" /></ConductingEquipment>
				<ConductingEquipment name="DIS_F2_B1" type="DIS" esld:x="21" esld:y="10" esld:lx="18.5" esld:ly="11">
					<Terminal esld:uuid="7d6b9ce7-3e29-4753-9bee-a7d7024a7df3" name="T2" connectivityNode="SS1/VL_50kV/BB1/L" substationName="SS1" voltageLevelName="VL_50kV" bayName="BB1" cNodeName="L" />
					<Terminal esld:uuid="a974066a-d475-4fc0-b889-581e59293832" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed2/L4" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed2" cNodeName="L4" />
				<LNode iedName="FEED2" ldInst="LD1" prefix="" lnClass="XSWI" lnInst="2" /><LNode iedName="FEED2" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="2" /></ConductingEquipment>
				<ConductingEquipment name="DIS_F2_B2" type="DIS" esld:x="23" esld:y="10" esld:lx="24" esld:ly="11">
					<Terminal esld:uuid="9ff9e493-4927-4fbc-bb0c-53ede0de840a" name="T2" connectivityNode="SS1/VL_50kV/BB2/L" substationName="SS1" voltageLevelName="VL_50kV" bayName="BB2" cNodeName="L" />
					<Terminal esld:uuid="fa657821-8507-4bee-befd-585cce64e81c" name="T1" connectivityNode="SS1/VL_50kV/Bay_Feed2/L4" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_Feed2" cNodeName="L4" />
				<LNode iedName="FEED2" ldInst="LD1" prefix="" lnClass="XSWI" lnInst="3" /><LNode iedName="FEED2" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="3" /></ConductingEquipment>
				<ConnectivityNode name="L4" pathName="SS1/VL_50kV/Bay_Feed2/L4">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="21.5" esld:y="10.16" esld:uuid="a974066a-d475-4fc0-b889-581e59293832" />
							<esld:Vertex esld:x="21.5" esld:y="9" />
							<esld:Vertex esld:x="22.5" esld:y="9" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="23.5" esld:y="10.16" esld:uuid="fa657821-8507-4bee-befd-585cce64e81c" />
							<esld:Vertex esld:x="23.5" esld:y="9" />
							<esld:Vertex esld:x="22.5" esld:y="9" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="22.5" esld:y="9" />
							<esld:Vertex esld:x="22.5" esld:y="8.84" esld:uuid="a9847d00-f82f-4cb5-ad1a-548f847f9a22" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
				<ConnectivityNode name="L3" pathName="SS1/VL_50kV/Bay_Feed2/L3">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="22.5" esld:y="6.84" esld:uuid="8c75a7b0-ec30-4f8a-b68b-f50796f928a2" />
							<esld:Vertex esld:x="22.5" esld:y="8.16" esld:uuid="aa48db86-949a-4642-8862-be3ce1c46f51" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
				<ConnectivityNode name="L2" pathName="SS1/VL_50kV/Bay_Feed2/L2">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="22.5" esld:y="4.84" esld:uuid="f0acbd4c-9cac-4a8c-a95b-ff53ff57b416" />
							<esld:Vertex esld:x="22.5" esld:y="6.16" esld:uuid="d0727ca4-059e-488f-8109-75c135b2af43" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
				<ConnectivityNode name="L1" pathName="SS1/VL_50kV/Bay_Feed2/L1">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="22.5" esld:y="2.84" esld:uuid="432c5857-8310-4a79-ac2f-f9247801e31c" />
							<esld:Vertex esld:x="22.5" esld:y="3.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="20.5" esld:y="4.16" esld:uuid="032e98a9-5a43-4a96-9581-0a1cb3ac0879" />
							<esld:Vertex esld:x="20.5" esld:y="3.5" />
							<esld:Vertex esld:x="22.5" esld:y="3.5" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="22.5" esld:y="3.5" />
							<esld:Vertex esld:x="22.5" esld:y="4.16" esld:uuid="708adebc-8c83-404a-a604-03e76735bba7" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
			</Bay>
			<Bay name="Bay_Busbar1" esld:x="3" esld:y="13" esld:lx="3" esld:ly="13" esld:w="62" esld:h="1">
				<ConductingEquipment name="VTR_B1" type="VTR" esld:x="7" esld:y="13" esld:lx="7" esld:ly="15" esld:rot="1">
					<Terminal esld:uuid="db2508d2-bf2f-47d5-aa65-dfed0290e141" name="T1" connectivityNode="SS1/VL_50kV/BB1/L" substationName="SS1" voltageLevelName="VL_50kV" bayName="BB1" cNodeName="L" />
				</ConductingEquipment>
			</Bay>
			<Bay name="Bay_Busbar2" esld:x="3" esld:y="15" esld:lx="3" esld:ly="17.5" esld:w="62" esld:h="1">
				<ConductingEquipment type="VTR" name="VTR_B2" esld:x="7" esld:y="15" esld:lx="7" esld:ly="17" esld:rot="1">
					<Terminal esld:uuid="02db15aa-549f-4300-9a3b-f5d94d92b8b8" name="T1" connectivityNode="SS1/VL_50kV/BB2/L" substationName="SS1" voltageLevelName="VL_50kV" bayName="BB2" cNodeName="L" />
				</ConductingEquipment>
			</Bay>
			<Bay name="Bay_PT1" esld:x="28" esld:y="17" esld:lx="28" esld:ly="17" esld:w="8" esld:h="7">
				<ConductingEquipment name="CTR_PT1" type="CTR" esld:x="32" esld:y="22" esld:lx="33" esld:ly="23">
					<Terminal esld:uuid="a469286e-2577-4e68-abc0-dea4fb0a6e37" name="T2" connectivityNode="SS1/VL_50kV/Bay_PT1/L3" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT1" cNodeName="L3" />
					<Terminal esld:uuid="27e9a550-621e-48d1-99f4-e8afa2df56ac" name="T1" connectivityNode="SS1/VL_50kV/Bay_PT1/L2" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT1" cNodeName="L2" />
				</ConductingEquipment>
				<ConductingEquipment name="CBR_PT1" type="CBR" esld:x="32" esld:y="20" esld:lx="33" esld:ly="21">
					<Terminal esld:uuid="42f748cb-5c61-4f93-b2d3-f406d1a505de" name="T2" connectivityNode="SS1/VL_50kV/Bay_PT1/L2" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT1" cNodeName="L2" />
					<Terminal esld:uuid="b010d0ad-65e4-487a-94e0-ee604ae52934" name="T1" connectivityNode="SS1/VL_50kV/Bay_PT1/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT1" cNodeName="L1" />
				<LNode iedName="TR1" ldInst="LD1" prefix="" lnClass="XCBR" lnInst="1" /><LNode iedName="TR1" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="1" /></ConductingEquipment>
				<ConductingEquipment name="DIS_PT1_B1" type="DIS" esld:x="31" esld:y="18" esld:lx="28.5" esld:ly="19">
					<Terminal esld:uuid="56b98e94-771a-4ccd-bf8e-033d12326780" name="T1" connectivityNode="SS1/VL_50kV/BB1/L" substationName="SS1" voltageLevelName="VL_50kV" bayName="BB1" cNodeName="L" />
					<Terminal esld:uuid="c62586d7-94b6-4ea9-bbc7-7bf9c61cdd2c" name="T2" connectivityNode="SS1/VL_50kV/Bay_PT1/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT1" cNodeName="L1" />
				<LNode iedName="TR1" ldInst="LD1" prefix="" lnClass="XSWI" lnInst="2" /><LNode iedName="TR1" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="2" /></ConductingEquipment>
				<ConductingEquipment name="DIS_PT1_B2" type="DIS" esld:x="33" esld:y="18" esld:lx="34" esld:ly="19">
					<Terminal esld:uuid="79e2a905-c383-4565-9e1b-68a49665497c" name="T1" connectivityNode="SS1/VL_50kV/BB2/L" substationName="SS1" voltageLevelName="VL_50kV" bayName="BB2" cNodeName="L" />
					<Terminal esld:uuid="d764c504-5824-4728-925d-d66ddeb941d6" name="T2" connectivityNode="SS1/VL_50kV/Bay_PT1/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT1" cNodeName="L1" />
				<LNode iedName="TR1" ldInst="LD1" prefix="" lnClass="XSWI" lnInst="3" /><LNode iedName="TR1" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="3" /></ConductingEquipment>
				<ConnectivityNode name="L3" pathName="SS1/VL_50kV/Bay_PT1/L3">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="32.5" esld:y="24.8" esld:uuid="ea1ac572-6f71-49fe-aa19-d18a327331cf" />
							<esld:Vertex esld:x="32.5" esld:y="22.84" esld:uuid="a469286e-2577-4e68-abc0-dea4fb0a6e37" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
				<ConnectivityNode name="L2" pathName="SS1/VL_50kV/Bay_PT1/L2">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="32.5" esld:y="20.84" esld:uuid="42f748cb-5c61-4f93-b2d3-f406d1a505de" />
							<esld:Vertex esld:x="32.5" esld:y="22.16" esld:uuid="27e9a550-621e-48d1-99f4-e8afa2df56ac" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
				<ConnectivityNode name="L1" pathName="SS1/VL_50kV/Bay_PT1/L1">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="31.5" esld:y="18.84" esld:uuid="c62586d7-94b6-4ea9-bbc7-7bf9c61cdd2c" />
							<esld:Vertex esld:x="31.5" esld:y="20" />
							<esld:Vertex esld:x="32.5" esld:y="20" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="33.5" esld:y="18.84" esld:uuid="d764c504-5824-4728-925d-d66ddeb941d6" />
							<esld:Vertex esld:x="33.5" esld:y="20" />
							<esld:Vertex esld:x="32.5" esld:y="20" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="32.5" esld:y="20" />
							<esld:Vertex esld:x="32.5" esld:y="20.16" esld:uuid="b010d0ad-65e4-487a-94e0-ee604ae52934" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
			</Bay>
			<Bay name="Bay_PT2" esld:x="38" esld:y="17" esld:lx="38" esld:ly="17" esld:w="8" esld:h="7">
				<ConductingEquipment type="DIS" name="DIS_PT2_B2" esld:x="42" esld:y="18" esld:lx="43" esld:ly="19">
					<Terminal esld:uuid="39812065-a0d8-403c-b747-c6a29dcc2386" name="T1" connectivityNode="SS1/VL_50kV/BB2/L" substationName="SS1" voltageLevelName="VL_50kV" bayName="BB2" cNodeName="L" />
					<Terminal esld:uuid="4e891b49-65ec-410c-97fa-b75b3f07df42" name="T2" connectivityNode="SS1/VL_50kV/Bay_PT2/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT2" cNodeName="L1" />
				<LNode iedName="TR2" ldInst="LD1" prefix="" lnClass="XSWI" lnInst="3" /><LNode iedName="TR2" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="3" /></ConductingEquipment>
				<ConductingEquipment type="DIS" name="DIS_PT2_B1" esld:x="40" esld:y="18" esld:lx="37.5" esld:ly="19">
					<Terminal esld:uuid="c97e5175-5090-45a1-a72c-fd86fbd36c1a" name="T1" connectivityNode="SS1/VL_50kV/BB1/L" substationName="SS1" voltageLevelName="VL_50kV" bayName="BB1" cNodeName="L" />
					<Terminal esld:uuid="89379edb-16e8-4e99-addf-136a4c449b6d" name="T2" connectivityNode="SS1/VL_50kV/Bay_PT2/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT2" cNodeName="L1" />
				<LNode iedName="TR2" ldInst="LD1" prefix="" lnClass="XSWI" lnInst="2" /><LNode iedName="TR2" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="2" /></ConductingEquipment>
				<ConductingEquipment type="CBR" name="CBR_PT2" esld:x="41" esld:y="20" esld:lx="42" esld:ly="21">
					<Terminal esld:uuid="e2618923-6714-49ba-a144-81e2d2cd39e3" name="T2" connectivityNode="SS1/VL_50kV/Bay_PT2/L2" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT2" cNodeName="L2" />
					<Terminal esld:uuid="7ff23123-03ee-4ca5-b504-41371aa92ab5" name="T1" connectivityNode="SS1/VL_50kV/Bay_PT2/L1" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT2" cNodeName="L1" />
				<LNode iedName="TR2" ldInst="LD1" prefix="" lnClass="XCBR" lnInst="1" /><LNode iedName="TR2" ldInst="LD1" prefix="" lnClass="CSWI" lnInst="1" /></ConductingEquipment>
				<ConductingEquipment name="CTR_PT2" type="CTR" esld:x="41" esld:y="22" esld:lx="42" esld:ly="23">
					<Terminal esld:uuid="494586ee-b986-43f7-ae20-a08c7fb7eaca" name="T2" connectivityNode="SS1/VL_50kV/Bay_PT2/L3" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT2" cNodeName="L3" />
					<Terminal esld:uuid="0c7c7e18-c9dc-42cb-be5b-0c30bb069987" name="T1" connectivityNode="SS1/VL_50kV/Bay_PT2/L2" substationName="SS1" voltageLevelName="VL_50kV" bayName="Bay_PT2" cNodeName="L2" />
				</ConductingEquipment>
				<ConnectivityNode name="L3" pathName="SS1/VL_50kV/Bay_PT2/L3">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="41.5" esld:y="24.8" esld:uuid="e9cb2c53-76ca-433e-af96-c590a0ca3f90" />
							<esld:Vertex esld:x="41.5" esld:y="22.84" esld:uuid="494586ee-b986-43f7-ae20-a08c7fb7eaca" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
				<ConnectivityNode name="L2" pathName="SS1/VL_50kV/Bay_PT2/L2">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="41.5" esld:y="20.84" esld:uuid="e2618923-6714-49ba-a144-81e2d2cd39e3" />
							<esld:Vertex esld:x="41.5" esld:y="22.16" esld:uuid="0c7c7e18-c9dc-42cb-be5b-0c30bb069987" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
				<ConnectivityNode name="L1" pathName="SS1/VL_50kV/Bay_PT2/L1">
					<Private type="Transpower-SLD-Vertices">
						<esld:Section>
							<esld:Vertex esld:x="40.5" esld:y="18.84" esld:uuid="89379edb-16e8-4e99-addf-136a4c449b6d" />
							<esld:Vertex esld:x="40.5" esld:y="20" />
							<esld:Vertex esld:x="41.5" esld:y="20" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="42.5" esld:y="18.84" esld:uuid="4e891b49-65ec-410c-97fa-b75b3f07df42" />
							<esld:Vertex esld:x="42.5" esld:y="20" />
							<esld:Vertex esld:x="41.5" esld:y="20" />
						</esld:Section>
						<esld:Section>
							<esld:Vertex esld:x="41.5" esld:y="20" />
							<esld:Vertex esld:x="41.5" esld:y="20.16" esld:uuid="7ff23123-03ee-4ca5-b504-41371aa92ab5" />
						</esld:Section>
					</Private>
				</ConnectivityNode>
			</Bay>
		</VoltageLevel>
		</Substation>
	<Communication>
		<SubNetwork name="substation_lan" desc="" type="8-MMS">
			<BitRate unit="b/s" multiplier="M">100</BitRate>
			<ConnectedAP iedName="FEED1" apName="AP1">
				<Address>
					<P type="IP">172.16.1.20</P>
					<P type="IP-SUBNET">255.255.255.0</P>
					<P type="OSI-TSEL">0001</P>
					<P type="OSI-SSEL">0001</P>
					<P type="OSI-PSEL">00000001</P>
					<P type="VLAN-ID">100</P>
					<P type="VLAN-PRIORITY">4</P>
				</Address>
				<GSE ldInst="LD1" cbName="Feed1SwiPosCtl">
					<Address>
						<P type="MAC-Address">01-0C-CD-01-00-20</P>
						<P type="APPID">0000</P>
						<P type="VLAN-ID">000</P>
						<P type="VLAN-PRIORITY">4</P>
					</Address>
					<MinTime unit="s" multiplier="m">10</MinTime>
					<MaxTime unit="s" multiplier="m">10000</MaxTime>
				</GSE>
			</ConnectedAP>
			<ConnectedAP iedName="FEED2" apName="AP1">
				<Address>
					<P type="IP">172.16.1.21</P>
					<P type="IP-SUBNET">255.255.255.0</P>
					<P type="OSI-TSEL">0001</P>
					<P type="OSI-SSEL">0001</P>
					<P type="OSI-PSEL">00000001</P>
					<P type="VLAN-ID">100</P>
					<P type="VLAN-PRIORITY">4</P>
				</Address>
				<GSE ldInst="LD1" cbName="Feed2SwiPosCtl">
					<Address>
						<P type="MAC-Address">01-0C-CD-01-00-21</P>
						<P type="APPID">0001</P>
						<P type="VLAN-ID">000</P>
						<P type="VLAN-PRIORITY">4</P>
					</Address>
					<MinTime unit="s" multiplier="m">10</MinTime>
					<MaxTime unit="s" multiplier="m">10000</MaxTime>
				</GSE>
			</ConnectedAP>
			<ConnectedAP iedName="BUS1" apName="AP1">
				<Address>
					<P type="IP">172.16.1.30</P>
					<P type="IP-SUBNET">255.255.255.0</P>
					<P type="OSI-TSEL">0001</P>
					<P type="OSI-SSEL">0001</P>
					<P type="OSI-PSEL">00000001</P>
					<P type="VLAN-ID">100</P>
					<P type="VLAN-PRIORITY">4</P>
				</Address>
				<GSE ldInst="LD1" cbName="Bus1TripEventCb">
					<Address>
						<P type="MAC-Address">01-0C-CD-01-00-30</P>
						<P type="APPID">0004</P>
						<P type="VLAN-ID">000</P>
						<P type="VLAN-PRIORITY">4</P>
					</Address>
					<MinTime unit="s" multiplier="m">10</MinTime>
					<MaxTime unit="s" multiplier="m">10000</MaxTime>
				</GSE>
			</ConnectedAP>
			<ConnectedAP iedName="BUS2" apName="AP1">
				<Address>
					<P type="IP">172.16.1.31</P>
					<P type="IP-SUBNET">255.255.255.0</P>
					<P type="OSI-TSEL">0001</P>
					<P type="OSI-SSEL">0001</P>
					<P type="OSI-PSEL">00000001</P>
					<P type="VLAN-ID">100</P>
					<P type="VLAN-PRIORITY">4</P>
				</Address>
				<GSE ldInst="LD1" cbName="Bus2TripEventCb">
					<Address>
						<P type="MAC-Address">01-0C-CD-01-00-31</P>
						<P type="APPID">0005</P>
						<P type="VLAN-ID">000</P>
						<P type="VLAN-PRIORITY">4</P>
					</Address>
					<MinTime unit="s" multiplier="m">10</MinTime>
					<MaxTime unit="s" multiplier="m">10000</MaxTime>
				</GSE>
			</ConnectedAP>
			<ConnectedAP iedName="TR1" apName="AP1">
				<Address>
					<P type="IP">172.16.1.40</P>
					<P type="IP-SUBNET">255.255.255.0</P>
					<P type="OSI-TSEL">0001</P>
					<P type="OSI-SSEL">0001</P>
					<P type="OSI-PSEL">00000001</P>
					<P type="VLAN-ID">100</P>
					<P type="VLAN-PRIORITY">4</P>
				</Address>
				<GSE ldInst="LD1" cbName="Trafo1SwiPosCtl">
					<Address>
						<P type="MAC-Address">01-0C-CD-01-00-40</P>
						<P type="APPID">0002</P>
						<P type="VLAN-ID">000</P>
						<P type="VLAN-PRIORITY">4</P>
					</Address>
					<MinTime unit="s" multiplier="m">10</MinTime>
					<MaxTime unit="s" multiplier="m">10000</MaxTime>
				</GSE>
			</ConnectedAP>
			<ConnectedAP iedName="TR2" apName="AP1">
				<Address>
					<P type="IP">172.16.1.41</P>
					<P type="IP-SUBNET">255.255.255.0</P>
					<P type="OSI-TSEL">0001</P>
					<P type="OSI-SSEL">0001</P>
					<P type="OSI-PSEL">00000001</P>
					<P type="VLAN-ID">100</P>
					<P type="VLAN-PRIORITY">4</P>
				</Address>
				<GSE ldInst="LD1" cbName="Trafo2SwiPosCtl">
					<Address>
						<P type="MAC-Address">01-0C-CD-01-00-41</P>
						<P type="APPID">0003</P>
						<P type="VLAN-ID">000</P>
						<P type="VLAN-PRIORITY">4</P>
					</Address>
					<MinTime unit="s" multiplier="m">10</MinTime>
					<MaxTime unit="s" multiplier="m">10000</MaxTime>
				</GSE>
			</ConnectedAP>
		</SubNetwork>
	</Communication>
	<IED name="FEED1" manufacturer="OpenSCD" esld:x="8" esld:y="7" esld:lx="8" esld:ly="7">
		<Services>
			<DynAssociation />
			<GetDirectory />
			<GetDataObjectDefinition />
			<GetDataSetValue />
			<DataSetDirectory />
			<ConfDataSet max="10" maxAttributes="50" />
			<DynDataSet max="10" />
			<ReadWrite />
			<ConfReportControl max="10" bufMode="both" bufConf="true" />
			<GetCBValues />
			<ConfLNs fixPrefix="true" fixLnInst="true" />
			<GOOSE max="5" />
			<GSSE max="5" />
			<FileHandling />
			<GSEDir />
		</Services>
		<AccessPoint name="AP1">
			<Server>
				<Authentication />
				<LDevice inst="LD1">
					<LN0 lnClass="LLN0" inst="" lnType="LLN0$oscd$_86a723959df7af2e">
						<DataSet name="Feed1SwiPos">
							<FCDA ldInst="LD1" prefix="" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="XSWI" lnInst="2" doName="Pos" daName="stVal" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="XSWI" lnInst="3" doName="Pos" daName="stVal" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="XSWI" lnInst="4" doName="Pos" daName="stVal" fc="ST" />
						</DataSet>
						<ReportControl name="ReportControl_001" buffered="true" rptID="Feed1Swi" bufTime="100" confRev="1" datSet="Feed1SwiPos" intgPd="2000">
							<TrgOps dchg="true" period="true" gi="true" />
							<OptFields seqNum="true" timeStamp="true" dataSet="true" reasonCode="true" configRef="true" />
							<RptEnabled max="1" />
						</ReportControl>
						<ReportControl name="ReportControl_002" buffered="true" rptID="Feed1Swi2" bufTime="100" confRev="1" datSet="Feed1SwiPos" intgPd="2000">
							<TrgOps dchg="true" period="true" gi="true" />
							<OptFields seqNum="true" timeStamp="true" dataSet="true" reasonCode="true" configRef="true" />
							<RptEnabled max="1" />
						</ReportControl>
						<GSEControl name="Feed1SwiPosCtl" confRev="0" type="GOOSE" appID="0001" datSet="Feed1SwiPos" />
					</LN0>
					<LN lnClass="LPHD" lnType="LPHD$oscd$_8ca3cb769327538e" inst="1" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="1" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="2" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="3" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="4" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="1" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="2" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="3" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="4" prefix="B2" />
					<LN lnClass="PTOC" lnType="PTOC$oscd$_72a47fe03fb65b2f" inst="1" prefix="">
						<DOI name="StrVal">
							<SDI name="setMag">
								<DAI name="f">
									<Val>100.3</Val>
								</DAI>
							</SDI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="PTOC_Amp1" desc="sample_a1" iedName="FEED1" ldInst="LD1" lnClass="TCTR" lnInst="1" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp2" desc="sample_a2" iedName="FEED1" ldInst="LD1" lnClass="TCTR" lnInst="2" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp3" desc="sample_a3" iedName="FEED1" ldInst="LD1" lnClass="TCTR" lnInst="3" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp4" desc="sample_a4" iedName="FEED1" ldInst="LD1" lnClass="TCTR" lnInst="4" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="1" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="FEED1" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED1" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Loc" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="2" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="FEED1" ldInst="LD1" lnClass="XSWI" lnInst="2" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED1" ldInst="LD1" lnClass="XSWI" lnInst="2" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="3" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="FEED1" ldInst="LD1" lnClass="XSWI" lnInst="3" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED1" ldInst="LD1" lnClass="XSWI" lnInst="3" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="4" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="FEED1" ldInst="LD1" lnClass="XSWI" lnInst="4" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED1" ldInst="LD1" lnClass="XSWI" lnInst="4" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" serviceType="GOOSE" iedName="BUS1" ldInst="LD1" lnClass="PTRC" lnInst="1" prefix="" doName="Tr" daName="general" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Bus1TripEventCn" />
							<ExtRef intAddr="Op2" serviceType="GOOSE" iedName="BUS2" ldInst="LD1" lnClass="PTRC" lnInst="1" prefix="" doName="Tr" daName="general" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Bus2TripEventCn" />
							<ExtRef intAddr="Op3" desc="PTOC_trip" iedName="FEED1" ldInst="LD1" lnClass="PTOC" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XCBR" lnType="XCBR$oscd$_420e2f9e111e8bf2" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED1" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="Tr" desc="Trip" iedName="FEED1" ldInst="LD1" lnClass="PTRC" lnInst="1" doName="Tr" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="FEED1" ldInst="LD1" lnClass="CSWI" lnInst="1" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="FEED1" ldInst="LD1" lnClass="CSWI" lnInst="1" doName="OpCls" daName="general" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="FEED1" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XSWI" lnType="XSWI$oscd$_430901aa6c264cc1" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED1" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="FEED1" ldInst="LD1" lnClass="CSWI" lnInst="2" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="FEED1" ldInst="LD1" lnClass="CSWI" lnInst="2" doName="OpCls" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XSWI" lnType="XSWI$oscd$_430901aa6c264cc1" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED1" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="FEED1" ldInst="LD1" lnClass="CSWI" lnInst="3" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="FEED1" ldInst="LD1" lnClass="CSWI" lnInst="3" doName="OpCls" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XSWI" lnType="XSWI$oscd$_430901aa6c264cc1" inst="4" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED1" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="FEED1" ldInst="LD1" lnClass="CSWI" lnInst="4" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="FEED1" ldInst="LD1" lnClass="CSWI" lnInst="4" doName="OpCls" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="FEED1" ldInst="LD1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="FEED1" ldInst="LD1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="FEED1" ldInst="LD1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="FEED1" ldInst="LD1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Vol1" desc="sample_v1" iedName="FEED1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="1" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol2" desc="sample_v2" iedName="FEED1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="2" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol3" desc="sample_v3" iedName="FEED1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="3" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol4" desc="sample_v4" iedName="FEED1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="4" doName="VolSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Vol1" desc="sample_v1" iedName="FEED1" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="1" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol2" desc="sample_v2" iedName="FEED1" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="2" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol3" desc="sample_v3" iedName="FEED1" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="3" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol4" desc="sample_v4" iedName="FEED1" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="4" doName="VolSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
				</LDevice>
			</Server>
		</AccessPoint>
	</IED>
	<IED name="FEED2" manufacturer="OpenSCD" esld:x="18" esld:y="7" esld:lx="18" esld:ly="7">
		<Services>
			<DynAssociation />
			<GetDirectory />
			<GetDataObjectDefinition />
			<GetDataSetValue />
			<DataSetDirectory />
			<ConfDataSet max="10" maxAttributes="50" />
			<DynDataSet max="10" />
			<ReadWrite />
			<ConfReportControl max="10" bufMode="both" bufConf="true" />
			<GetCBValues />
			<ConfLNs fixPrefix="true" fixLnInst="true" />
			<GOOSE max="5" />
			<GSSE max="5" />
			<FileHandling />
			<GSEDir />
		</Services>
		<AccessPoint name="AP1">
			<Server>
				<Authentication />
				<LDevice inst="LD1">
					<LN0 lnClass="LLN0" inst="" lnType="LLN0$oscd$_86a723959df7af2e">
						<DataSet name="Feed2SwiPos">
							<FCDA ldInst="LD1" prefix="" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="XSWI" lnInst="2" doName="Pos" daName="stVal" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="XSWI" lnInst="3" doName="Pos" daName="stVal" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="XSWI" lnInst="4" doName="Pos" daName="stVal" fc="ST" />
						</DataSet>
						<ReportControl name="ReportControl_001" buffered="true" rptID="Feed2Swi" bufTime="100" confRev="1" datSet="Feed2SwiPos" intgPd="2000">
							<TrgOps dchg="true" period="true" gi="true" />
							<OptFields seqNum="true" timeStamp="true" dataSet="true" reasonCode="true" configRef="true" />
							<RptEnabled max="1" />
						</ReportControl>
						<ReportControl name="ReportControl_002" buffered="true" rptID="Feed2Swi2" bufTime="100" confRev="1" datSet="Feed2SwiPos" intgPd="2000">
							<TrgOps dchg="true" period="true" gi="true" />
							<OptFields seqNum="true" timeStamp="true" dataSet="true" reasonCode="true" configRef="true" />
							<RptEnabled max="1" />
						</ReportControl>
						<GSEControl name="Feed2SwiPosCtl" confRev="0" type="GOOSE" appID="0002" datSet="Feed2SwiPos" />
					</LN0>
					<LN lnClass="LPHD" lnType="LPHD$oscd$_8ca3cb769327538e" inst="1" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="1" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="2" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="3" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="4" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="1" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="2" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="3" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="4" prefix="B2" />
					<LN lnClass="PTOC" lnType="PTOC$oscd$_72a47fe03fb65b2f" inst="1" prefix="">
						<DOI name="StrVal">
							<SDI name="setMag">
								<DAI name="f">
									<Val>42.3</Val>
								</DAI>
							</SDI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="PTOC_Amp1" desc="sample_a1" iedName="FEED2" ldInst="LD1" lnClass="TCTR" lnInst="1" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp2" desc="sample_a2" iedName="FEED2" ldInst="LD1" lnClass="TCTR" lnInst="2" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp3" desc="sample_a3" iedName="FEED2" ldInst="LD1" lnClass="TCTR" lnInst="3" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp4" desc="sample_a4" iedName="FEED2" ldInst="LD1" lnClass="TCTR" lnInst="4" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="1" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED2" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="FEED2" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="2" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED2" ldInst="LD1" lnClass="XSWI" lnInst="2" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="FEED2" ldInst="LD1" lnClass="XSWI" lnInst="2" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="3" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED2" ldInst="LD1" lnClass="XSWI" lnInst="3" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="FEED2" ldInst="LD1" lnClass="XSWI" lnInst="3" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="4" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED2" ldInst="LD1" lnClass="XSWI" lnInst="4" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="FEED2" ldInst="LD1" lnClass="XSWI" lnInst="4" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" serviceType="GOOSE" iedName="BUS1" ldInst="LD1" lnClass="PTRC" lnInst="2" prefix="" doName="Tr" daName="general" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Bus1TripEventCn" />
							<ExtRef intAddr="Op2" serviceType="GOOSE" iedName="BUS2" ldInst="LD1" lnClass="PTRC" lnInst="2" prefix="" doName="Tr" daName="general" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Bus2TripEventCn" />
							<ExtRef intAddr="Op3" desc="PTOC_trip" iedName="FEED2" ldInst="LD1" lnClass="PTOC" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XCBR" lnType="XCBR$oscd$_420e2f9e111e8bf2" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED2" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="Tr" desc="Trip" iedName="FEED2" ldInst="LD1" lnClass="PTRC" lnInst="1" doName="Tr" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="FEED2" ldInst="LD1" lnClass="CSWI" lnInst="1" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="FEED2" ldInst="LD1" lnClass="CSWI" lnInst="1" doName="OpCls" daName="general" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="FEED2" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XSWI" lnType="XSWI$oscd$_430901aa6c264cc1" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED2" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="FEED2" ldInst="LD1" lnClass="CSWI" lnInst="2" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="FEED2" ldInst="LD1" lnClass="CSWI" lnInst="2" doName="OpCls" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XSWI" lnType="XSWI$oscd$_430901aa6c264cc1" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED2" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="FEED2" ldInst="LD1" lnClass="CSWI" lnInst="3" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="FEED2" ldInst="LD1" lnClass="CSWI" lnInst="3" doName="OpCls" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XSWI" lnType="XSWI$oscd$_430901aa6c264cc1" inst="4" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="FEED2" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="FEED2" ldInst="LD1" lnClass="CSWI" lnInst="4" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="FEED2" ldInst="LD1" lnClass="CSWI" lnInst="4" doName="OpCls" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="FEED2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="FEED2" ldInst="LD1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="FEED2" ldInst="LD1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="FEED2" ldInst="LD1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="FEED2" ldInst="LD1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Vol1" desc="sample_v1" iedName="FEED2" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="1" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol2" desc="sample_v2" iedName="FEED2" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="2" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol3" desc="sample_v3" iedName="FEED2" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="3" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol4" desc="sample_v4" iedName="FEED2" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="4" doName="VolSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Vol1" desc="sample_v1" iedName="FEED2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="1" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol2" desc="sample_v2" iedName="FEED2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="2" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol3" desc="sample_v3" iedName="FEED2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="3" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol4" desc="sample_v4" iedName="FEED2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="4" doName="VolSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
				</LDevice>
			</Server>
		</AccessPoint>
	</IED>
	<IED name="BUS1" manufacturer="OpenSCD" esld:x="4" esld:y="13" esld:lx="4" esld:ly="13">
		<Services>
			<DynAssociation />
			<GetDirectory />
			<GetDataObjectDefinition />
			<GetDataSetValue />
			<DataSetDirectory />
			<ConfDataSet max="10" maxAttributes="50" />
			<DynDataSet max="10" />
			<ReadWrite />
			<GetCBValues />
			<ConfLNs fixPrefix="true" fixLnInst="true" />
			<GOOSE max="5" />
			<GSSE max="5" />
			<FileHandling />
			<GSEDir />
		</Services>
		<AccessPoint name="AP1">
			<Server>
				<Authentication />
				<LDevice inst="LD1">
					<LN0 lnClass="LLN0" inst="" lnType="LLN0$oscd$_86a723959df7af2e">
						<DataSet name="Bus1TripEvent">
							<FCDA ldInst="LD1" prefix="" lnClass="PDIF" lnInst="1" doName="Op" daName="general" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="PTRC" lnInst="1" doName="Tr" daName="general" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="PTRC" lnInst="2" doName="Tr" daName="general" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="PTRC" lnInst="3" doName="Tr" daName="general" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="PTRC" lnInst="4" doName="Tr" daName="general" fc="ST" />
						</DataSet>
						<GSEControl name="Bus1TripEventCb" confRev="0" type="GOOSE" appID="0003" datSet="Bus1TripEvent" />
					</LN0>
					<LN lnClass="LPHD" lnType="LPHD$oscd$_8ca3cb769327538e" inst="1" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="F1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="F1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="F1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="F1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="F2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="F2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="F2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="F2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="TR1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="TR1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="TR1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="TR1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="TR2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="TR2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="TR2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="TR2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="1" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="2" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="3" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="4" prefix="B1" />
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="BUS1" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="BUS1" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="BUS1" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="BUS1" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="BUS1" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="BUS1" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="BUS1" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="BUS1" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="BUS1" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="BUS1" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="BUS1" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="BUS1" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="4" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="BUS1" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="BUS1" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="BUS1" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="BUS1" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="5" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Vol1" desc="sample_v1" iedName="BUS1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="1" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol2" desc="sample_v2" iedName="BUS1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="2" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol3" desc="sample_v3" iedName="BUS1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="3" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol4" desc="sample_v4" iedName="BUS1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="4" doName="VolSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>

					<LN lnClass="PDIF" lnType="PDIF$oscd$_ab16057220381a95" inst="1" prefix="">
						<DOI name="LoSet">
							<SDI name="setMag">
								<DAI name="f">
									<Val>200.3</Val>
								</DAI>
							</SDI>
						</DOI>
						<DOI name="HiSet">
							<SDI name="setMag">
								<DAI name="f">
									<Val>0.3</Val>
								</DAI>
							</SDI>
						</DOI>
						<DOI name="MinOpTmms">
							<DAI name="setVal">
								<Val>42</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="PDIF_in_Amp1_0" desc="sample_a1" iedName="BUS1" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp2_0" desc="sample_a2" iedName="BUS1" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp3_0" desc="sample_a3" iedName="BUS1" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp4_0" desc="sample_a4" iedName="BUS1" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp1_1" desc="sample_a1" iedName="BUS1" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp2_1" desc="sample_a2" iedName="BUS1" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp3_1" desc="sample_a3" iedName="BUS1" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp4_1" desc="sample_a4" iedName="BUS1" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp1_0" desc="sample_a1" iedName="BUS1" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp2_0" desc="sample_a2" iedName="BUS1" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp3_0" desc="sample_a3" iedName="BUS1" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp4_0" desc="sample_a4" iedName="BUS1" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp1_1" desc="sample_a1" iedName="BUS1" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp2_1" desc="sample_a2" iedName="BUS1" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp3_1" desc="sample_a3" iedName="BUS1" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp4_1" desc="sample_a4" iedName="BUS1" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />

							<ExtRef intAddr="PDIF_in_swi_0" serviceType="GOOSE" iedName="FEED1" ldInst="LD1" lnClass="XSWI" lnInst="3" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Feed1SwiPosCtl" />
							<ExtRef intAddr="PDIF_in_swi_1" serviceType="GOOSE" iedName="FEED2" ldInst="LD1" lnClass="XSWI" lnInst="3" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Feed2SwiPosCtl" />
							<ExtRef intAddr="PDIF_out_swi_0" serviceType="GOOSE" iedName="TR1" ldInst="LD1" lnClass="XSWI" lnInst="2" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Trafo1SwiPosCtl" />
							<ExtRef intAddr="PDIF_out_swi_1" serviceType="GOOSE" iedName="TR2" ldInst="LD1" lnClass="XSWI" lnInst="2" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Trafo2SwiPosCtl" />

						</Inputs>
					</LN>

					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="in_nor_0" serviceType="GOOSE" iedName="FEED1" ldInst="LD1" lnClass="XSWI" lnInst="3" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Feed1SwiPosCtl" />
						</Inputs>
					</LN>
					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="in_nor_0" serviceType="GOOSE" iedName="FEED2" ldInst="LD1" lnClass="XSWI" lnInst="3" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Feed2SwiPosCtl" />
						</Inputs>
					</LN>
					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="in_nor_0" serviceType="GOOSE" iedName="TR1" ldInst="LD1" lnClass="XSWI" lnInst="2" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Trafo1SwiPosCtl" />
						</Inputs>
					</LN>
					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="4" prefix="">
						<Inputs>
							<ExtRef intAddr="in_nor_0" serviceType="GOOSE" iedName="TR2" ldInst="LD1" lnClass="XSWI" lnInst="2" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Trafo2SwiPosCtl" />
						</Inputs>
					</LN>

					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" desc="PDIF_trip" iedName="BUS1" ldInst="LD1" lnClass="PDIF" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="CILO_EnaOpn" iedName="BUS1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" desc="PDIF_trip" iedName="BUS1" ldInst="LD1" lnClass="PDIF" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="CILO_EnaOpn" iedName="BUS1" ldInst="LD1" lnClass="CILO" lnInst="2" doName="EnaOpn" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" desc="PDIF_trip" iedName="BUS1" ldInst="LD1" lnClass="PDIF" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="CILO_EnaOpn" iedName="BUS1" ldInst="LD1" lnClass="CILO" lnInst="3" doName="EnaOpn" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="4" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" desc="PDIF_trip" iedName="BUS1" ldInst="LD1" lnClass="PDIF" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="CILO_EnaOpn" iedName="BUS1" ldInst="LD1" lnClass="CILO" lnInst="4" doName="EnaOpn" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>

				</LDevice>
			</Server>
		</AccessPoint>
	</IED>
	<IED name="BUS2" manufacturer="OpenSCD" esld:x="3" esld:y="15" esld:lx="3" esld:ly="15">
		<Services>
			<DynAssociation />
			<GetDirectory />
			<GetDataObjectDefinition />
			<GetDataSetValue />
			<DataSetDirectory />
			<ConfDataSet max="10" maxAttributes="50" />
			<DynDataSet max="10" />
			<ReadWrite />
			<GetCBValues />
			<ConfLNs fixPrefix="true" fixLnInst="true" />
			<GOOSE max="5" />
			<GSSE max="5" />
			<FileHandling />
			<GSEDir />
		</Services>
		<AccessPoint name="AP1">
			<Server>
				<Authentication />
				<LDevice inst="LD1">
					<LN0 lnClass="LLN0" inst="" lnType="LLN0$oscd$_86a723959df7af2e">
						<DataSet name="Bus2TripEvent">
							<FCDA ldInst="LD1" prefix="" lnClass="PDIF" lnInst="1" doName="Op" daName="general" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="PTRC" lnInst="1" doName="Tr" daName="general" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="PTRC" lnInst="2" doName="Tr" daName="general" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="PTRC" lnInst="3" doName="Tr" daName="general" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="PTRC" lnInst="4" doName="Tr" daName="general" fc="ST" />
						</DataSet>
						<GSEControl name="Bus2TripEventCb" confRev="0" type="GOOSE" appID="0003" datSet="Bus2TripEvent" />
					</LN0>
					<LN lnClass="LPHD" lnType="LPHD$oscd$_8ca3cb769327538e" inst="1" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="F1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="F1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="F1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="F1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="F2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="F2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="F2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="F2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="TR1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="TR1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="TR1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="TR1" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="TR2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="TR2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="TR2" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="TR2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="1" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="2" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="3" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="4" prefix="B2" />
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="BUS2" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="BUS2" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="BUS2" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="BUS2" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="BUS2" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="BUS2" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="BUS2" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="BUS2" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="BUS2" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="BUS2" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="BUS2" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="BUS2" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="4" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="BUS2" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="BUS2" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="BUS2" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="BUS2" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="5" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Vol1" desc="sample_v1" iedName="BUS2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="1" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol2" desc="sample_v2" iedName="BUS2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="2" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol3" desc="sample_v3" iedName="BUS2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="3" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol4" desc="sample_v4" iedName="BUS2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="4" doName="VolSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>

					<LN lnClass="PDIF" lnType="PDIF$oscd$_ab16057220381a95" inst="1" prefix="">
						<DOI name="LoSet">
							<SDI name="setMag">
								<DAI name="f">
									<Val>200.5</Val>
								</DAI>
							</SDI>
						</DOI>
						<DOI name="HiSet">
							<SDI name="setMag">
								<DAI name="f">
									<Val>0.35</Val>
								</DAI>
							</SDI>
						</DOI>
						<DOI name="MinOpTmms">
							<DAI name="setVal">
								<Val>43</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="PDIF_in_Amp1_0" desc="sample_a1" iedName="BUS2" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp2_0" desc="sample_a2" iedName="BUS2" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp3_0" desc="sample_a3" iedName="BUS2" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp4_0" desc="sample_a4" iedName="BUS2" ldInst="LD1" prefix="F1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp1_1" desc="sample_a1" iedName="BUS2" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp2_1" desc="sample_a2" iedName="BUS2" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp3_1" desc="sample_a3" iedName="BUS2" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_in_Amp4_1" desc="sample_a4" iedName="BUS2" ldInst="LD1" prefix="F2" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp1_0" desc="sample_a1" iedName="BUS2" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp2_0" desc="sample_a2" iedName="BUS2" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp3_0" desc="sample_a3" iedName="BUS2" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp4_0" desc="sample_a4" iedName="BUS2" ldInst="LD1" prefix="TR1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp1_1" desc="sample_a1" iedName="BUS2" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp2_1" desc="sample_a2" iedName="BUS2" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp3_1" desc="sample_a3" iedName="BUS2" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="PDIF_out_Amp4_1" desc="sample_a4" iedName="BUS2" ldInst="LD1" prefix="TR2" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />

							<ExtRef intAddr="PDIF_in_swi_0" serviceType="GOOSE" iedName="FEED1" ldInst="LD1" lnClass="XSWI" lnInst="4" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Feed1SwiPosCtl" />
							<ExtRef intAddr="PDIF_in_swi_1" serviceType="GOOSE" iedName="FEED2" ldInst="LD1" lnClass="XSWI" lnInst="4" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Feed2SwiPosCtl" />
							<ExtRef intAddr="PDIF_out_swi_0" serviceType="GOOSE" iedName="TR1" ldInst="LD1" lnClass="XSWI" lnInst="3" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Trafo1SwiPosCtl" />
							<ExtRef intAddr="PDIF_out_swi_1" serviceType="GOOSE" iedName="TR2" ldInst="LD1" lnClass="XSWI" lnInst="3" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Trafo2SwiPosCtl" />
						</Inputs>
					</LN>

					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="in_nor_0" serviceType="GOOSE" iedName="FEED1" ldInst="LD1" lnClass="XSWI" lnInst="4" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Feed1SwiPosCtl" />
						</Inputs>
					</LN>
					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="in_nor_0" serviceType="GOOSE" iedName="FEED2" ldInst="LD1" lnClass="XSWI" lnInst="4" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Feed2SwiPosCtl" />
						</Inputs>
					</LN>
					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="in_nor_0" serviceType="GOOSE" iedName="TR1" ldInst="LD1" lnClass="XSWI" lnInst="3" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Trafo1SwiPosCtl" />
						</Inputs>
					</LN>
					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="4" prefix="">
						<Inputs>
							<ExtRef intAddr="in_nor_0" serviceType="GOOSE" iedName="TR2" ldInst="LD1" lnClass="XSWI" lnInst="3" prefix="" doName="Pos" daName="stVal" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Trafo2SwiPosCtl" />
						</Inputs>
					</LN>

					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" desc="PDIF_trip" iedName="BUS2" ldInst="LD1" lnClass="PDIF" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="CILO_EnaOpn" iedName="BUS2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" desc="PDIF_trip" iedName="BUS2" ldInst="LD1" lnClass="PDIF" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="CILO_EnaOpn" iedName="BUS2" ldInst="LD1" lnClass="CILO" lnInst="2" doName="EnaOpn" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" desc="PDIF_trip" iedName="BUS2" ldInst="LD1" lnClass="PDIF" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="CILO_EnaOpn" iedName="BUS2" ldInst="LD1" lnClass="CILO" lnInst="3" doName="EnaOpn" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="4" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" desc="PDIF_trip" iedName="BUS2" ldInst="LD1" lnClass="PDIF" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="CILO_EnaOpn" iedName="BUS2" ldInst="LD1" lnClass="CILO" lnInst="4" doName="EnaOpn" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>

				</LDevice>
			</Server>
		</AccessPoint>
	</IED>
	<IED name="TR1" manufacturer="OpenSCD" esld:x="28" esld:y="21" esld:lx="28" esld:ly="21">
		<Services>
			<DynAssociation />
			<GetDirectory />
			<GetDataObjectDefinition />
			<GetDataSetValue />
			<DataSetDirectory />
			<ConfDataSet max="10" maxAttributes="50" />
			<DynDataSet max="10" />
			<ReadWrite />
			<ConfReportControl max="10" bufMode="both" bufConf="true" />
			<GetCBValues />
			<ConfLNs fixPrefix="true" fixLnInst="true" />
			<GOOSE max="5" />
			<GSSE max="5" />
			<FileHandling />
			<GSEDir />
		</Services>
		<AccessPoint name="AP1">
			<Server>
				<Authentication />
				<LDevice inst="LD1">
					<LN0 lnClass="LLN0" inst="" lnType="LLN0$oscd$_86a723959df7af2e">
						<DataSet name="Trafo1SwiPos">
							<FCDA ldInst="LD1" prefix="" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="XSWI" lnInst="2" doName="Pos" daName="stVal" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="XSWI" lnInst="3" doName="Pos" daName="stVal" fc="ST" />
						</DataSet>
						<ReportControl name="ReportControl_001" buffered="true" rptID="Trafo1Swi" bufTime="100" confRev="1" datSet="Trafo1SwiPos" intgPd="2000">
							<TrgOps dchg="true" period="true" gi="true" />
							<OptFields seqNum="true" timeStamp="true" dataSet="true" reasonCode="true" configRef="true" />
							<RptEnabled max="1" />
						</ReportControl>
						<ReportControl name="ReportControl_002" buffered="true" rptID="Trafo1Swi2" bufTime="100" confRev="1" datSet="Trafo1SwiPos" intgPd="2000">
							<TrgOps dchg="true" period="true" gi="true" />
							<OptFields seqNum="true" timeStamp="true" dataSet="true" reasonCode="true" configRef="true" />
							<RptEnabled max="1" />
						</ReportControl>
						<GSEControl name="Trafo1SwiPosCtl" confRev="0" type="GOOSE" appID="0005" datSet="Trafo1SwiPos" />
					</LN0>
					<LN lnClass="LPHD" lnType="LPHD$oscd$_8ca3cb769327538e" inst="1" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="1" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="2" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="3" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="4" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="1" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="2" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="3" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="4" prefix="B2" />
					<LN lnClass="PTOC" lnType="PTOC$oscd$_72a47fe03fb65b2f" inst="1" prefix="">
						<DOI name="StrVal">
							<SDI name="setMag">
								<DAI name="f">
									<Val>100.3</Val>
								</DAI>
							</SDI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="PTOC_Amp1" desc="sample_a1" iedName="TR1" ldInst="LD1" lnClass="TCTR" lnInst="1" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp2" desc="sample_a2" iedName="TR1" ldInst="LD1" lnClass="TCTR" lnInst="2" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp3" desc="sample_a3" iedName="TR1" ldInst="LD1" lnClass="TCTR" lnInst="3" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp4" desc="sample_a4" iedName="TR1" ldInst="LD1" lnClass="TCTR" lnInst="4" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="1" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR1" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="TR1" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="2" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR1" ldInst="LD1" lnClass="XSWI" lnInst="2" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="TR1" ldInst="LD1" lnClass="XSWI" lnInst="2" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="TR1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="TR1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="3" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR1" ldInst="LD1" lnClass="XSWI" lnInst="3" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="TR1" ldInst="LD1" lnClass="XSWI" lnInst="3" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="TR1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="TR1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" serviceType="GOOSE" iedName="BUS1" ldInst="LD1" lnClass="PTRC" lnInst="3" prefix="" doName="Tr" daName="general" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Bus1TripEventCn" />
							<ExtRef intAddr="Op2" serviceType="GOOSE" iedName="BUS2" ldInst="LD1" lnClass="PTRC" lnInst="3" prefix="" doName="Tr" daName="general" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Bus2TripEventCn" />
							<ExtRef intAddr="Op3" desc="PTOC_trip" iedName="TR1" ldInst="LD1" lnClass="PTOC" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XCBR" lnType="XCBR$oscd$_420e2f9e111e8bf2" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR1" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="Tr" desc="Trip" iedName="TR1" ldInst="LD1" lnClass="PTRC" lnInst="1" doName="Tr" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="TR1" ldInst="LD1" lnClass="CSWI" lnInst="1" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="TR1" ldInst="LD1" lnClass="CSWI" lnInst="1" doName="OpCls" daName="general" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="TR1" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XSWI" lnType="XSWI$oscd$_430901aa6c264cc1" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR1" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="TR1" ldInst="LD1" lnClass="CSWI" lnInst="2" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="TR1" ldInst="LD1" lnClass="CSWI" lnInst="2" doName="OpCls" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="TR1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="TR1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XSWI" lnType="XSWI$oscd$_430901aa6c264cc1" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR1" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="TR1" ldInst="LD1" lnClass="CSWI" lnInst="3" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="TR1" ldInst="LD1" lnClass="CSWI" lnInst="3" doName="OpCls" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="TR1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="TR1" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="TR1" ldInst="LD1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="TR1" ldInst="LD1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="TR1" ldInst="LD1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="TR1" ldInst="LD1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Vol1" desc="sample_v1" iedName="TR1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="1" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol2" desc="sample_v2" iedName="TR1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="2" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol3" desc="sample_v3" iedName="TR1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="3" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol4" desc="sample_v4" iedName="TR1" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="4" doName="VolSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Vol1" desc="sample_v1" iedName="TR1" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="1" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol2" desc="sample_v2" iedName="TR1" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="2" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol3" desc="sample_v3" iedName="TR1" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="3" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol4" desc="sample_v4" iedName="TR1" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="4" doName="VolSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
				</LDevice>
			</Server>
		</AccessPoint>
	</IED>
	<IED name="TR2" manufacturer="OpenSCD" esld:x="38" esld:y="21" esld:lx="38" esld:ly="21">
		<Services>
			<DynAssociation />
			<GetDirectory />
			<GetDataObjectDefinition />
			<GetDataSetValue />
			<DataSetDirectory />
			<ConfDataSet max="10" maxAttributes="50" />
			<DynDataSet max="10" />
			<ReadWrite />
			<ConfReportControl max="10" bufMode="both" bufConf="true" />
			<GetCBValues />
			<ConfLNs fixPrefix="true" fixLnInst="true" />
			<GOOSE max="5" />
			<GSSE max="5" />
			<FileHandling />
			<GSEDir />
		</Services>
		<AccessPoint name="AP1">
			<Server>
				<Authentication />
				<LDevice inst="LD1">
					<LN0 lnClass="LLN0" inst="" lnType="LLN0$oscd$_86a723959df7af2e">
						<DataSet name="Trafo2SwiPos">
							<FCDA ldInst="LD1" prefix="" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="XSWI" lnInst="2" doName="Pos" daName="stVal" fc="ST" />
							<FCDA ldInst="LD1" prefix="" lnClass="XSWI" lnInst="3" doName="Pos" daName="stVal" fc="ST" />
						</DataSet>
						<ReportControl name="ReportControl_001" buffered="true" rptID="Trafo2Swi" bufTime="100" confRev="1" datSet="Trafo2SwiPos" intgPd="2000">
							<TrgOps dchg="true" period="true" gi="true" />
							<OptFields seqNum="true" timeStamp="true" dataSet="true" reasonCode="true" configRef="true" />
							<RptEnabled max="1" />
						</ReportControl>
						<ReportControl name="ReportControl_002" buffered="true" rptID="Trafo2Swi2" bufTime="100" confRev="1" datSet="Trafo2SwiPos" intgPd="2000">
							<TrgOps dchg="true" period="true" gi="true" />
							<OptFields seqNum="true" timeStamp="true" dataSet="true" reasonCode="true" configRef="true" />
							<RptEnabled max="1" />
						</ReportControl>
						<GSEControl name="Trafo2SwiPosCtl" confRev="0" type="GOOSE" appID="0006" datSet="Trafo2SwiPos" />
					</LN0>
					<LN lnClass="LPHD" lnType="LPHD$oscd$_8ca3cb769327538e" inst="1" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="1" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="2" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="3" prefix="" />
					<LN lnClass="TCTR" lnType="TCTR$oscd$_5b652afdf7c13340" inst="4" prefix="" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="1" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="2" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="3" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="4" prefix="B1" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="1" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="2" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="3" prefix="B2" />
					<LN lnClass="TVTR" lnType="TVTR$oscd$_e990c540f6cd3155" inst="4" prefix="B2" />
					<LN lnClass="PTOC" lnType="PTOC$oscd$_72a47fe03fb65b2f" inst="1" prefix="">
						<DOI name="StrVal">
							<SDI name="setMag">
								<DAI name="f">
									<Val>42.3</Val>
								</DAI>
							</SDI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="PTOC_Amp1" desc="sample_a1" iedName="TR2" ldInst="LD1" lnClass="TCTR" lnInst="1" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp2" desc="sample_a2" iedName="TR2" ldInst="LD1" lnClass="TCTR" lnInst="2" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp3" desc="sample_a3" iedName="TR2" ldInst="LD1" lnClass="TCTR" lnInst="3" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
							<ExtRef intAddr="PTOC_Amp4" desc="sample_a4" iedName="TR2" ldInst="LD1" lnClass="TCTR" lnInst="4" doName="AmpSv" serviceType="Poll" daName="instMag.i" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="1" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR2" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="TR2" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="2" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR2" ldInst="LD1" lnClass="XSWI" lnInst="2" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="TR2" ldInst="LD1" lnClass="XSWI" lnInst="2" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="TR2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="TR2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CSWI" lnType="CSWI$oscd$_a89dcc3919082155" inst="3" prefix="">
						<DOI name="Pos">
							<DAI name="ctlModel">
								<Val>sbo-with-enhanced-security</Val>
							</DAI>
						</DOI>
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR2" ldInst="LD1" lnClass="XSWI" lnInst="3" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="TR2" ldInst="LD1" lnClass="XSWI" lnInst="3" doName="Pos" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="TR2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="TR2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="PTRC" lnType="PTRC$oscd$_a60344357bbc2c12" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="Op1" serviceType="GOOSE" iedName="BUS1" ldInst="LD1" lnClass="PTRC" lnInst="4" prefix="" doName="Tr" daName="general" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Bus1TripEventCn" />
							<ExtRef intAddr="Op2" serviceType="GOOSE" iedName="BUS2" ldInst="LD1" lnClass="PTRC" lnInst="4" prefix="" doName="Tr" daName="general" srcLDInst="LD1" srcPrefix="" srcLNClass="LLN0" srcCBName="Bus2TripEventCn" />
							<ExtRef intAddr="Op3" desc="PTOC_trip" iedName="TR2" ldInst="LD1" lnClass="PTOC" lnInst="1" doName="Op" daName="general" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XCBR" lnType="XCBR$oscd$_420e2f9e111e8bf2" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR2" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="Tr" desc="Trip" iedName="TR2" ldInst="LD1" lnClass="PTRC" lnInst="1" doName="Tr" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="TR2" ldInst="LD1" lnClass="CSWI" lnInst="1" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="TR2" ldInst="LD1" lnClass="CSWI" lnInst="1" doName="OpCls" daName="general" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="CILO" lnType="CILO$oscd$_aa7ec79ef27309b1" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="stval" desc="xcbr_status" iedName="TR2" ldInst="LD1" lnClass="XCBR" lnInst="1" doName="Pos" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XSWI" lnType="XSWI$oscd$_430901aa6c264cc1" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR2" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="TR2" ldInst="LD1" lnClass="CSWI" lnInst="2" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="TR2" ldInst="LD1" lnClass="CSWI" lnInst="2" doName="OpCls" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="TR2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="TR2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="XSWI" lnType="XSWI$oscd$_430901aa6c264cc1" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="Loc" desc="Local_Remote" iedName="TR2" ldInst="LD1" lnClass="LLN0" doName="Loc" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="OpOpn" desc="Operate" iedName="TR2" ldInst="LD1" lnClass="CSWI" lnInst="3" doName="OpOpn" daName="general" serviceType="Poll" />
							<ExtRef intAddr="OpCls" desc="Operate" iedName="TR2" ldInst="LD1" lnClass="CSWI" lnInst="3" doName="OpCls" daName="general" serviceType="Poll" />
							<ExtRef intAddr="EnaOpn" desc="interlock_open" iedName="TR2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaOpn" daName="stVal" serviceType="Poll" />
							<ExtRef intAddr="EnaCls" desc="interlock_close" iedName="TR2" ldInst="LD1" lnClass="CILO" lnInst="1" doName="EnaCls" daName="stVal" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="1" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Amp1" desc="sample_a1" iedName="TR2" ldInst="LD1" lnClass="TCTR" lnInst="1" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp2" desc="sample_a2" iedName="TR2" ldInst="LD1" lnClass="TCTR" lnInst="2" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp3" desc="sample_a3" iedName="TR2" ldInst="LD1" lnClass="TCTR" lnInst="3" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Amp4" desc="sample_a4" iedName="TR2" ldInst="LD1" lnClass="TCTR" lnInst="4" doName="AmpSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="2" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Vol1" desc="sample_v1" iedName="TR2" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="1" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol2" desc="sample_v2" iedName="TR2" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="2" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol3" desc="sample_v3" iedName="TR2" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="3" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol4" desc="sample_v4" iedName="TR2" ldInst="LD1" prefix="B1" lnClass="TVTR" lnInst="4" doName="VolSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
					<LN lnClass="MMXU" lnType="MMXU$oscd$_3abca1c29963576c" inst="3" prefix="">
						<Inputs>
							<ExtRef intAddr="MMXU_Vol1" desc="sample_v1" iedName="TR2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="1" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol2" desc="sample_v2" iedName="TR2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="2" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol3" desc="sample_v3" iedName="TR2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="3" doName="VolSv" daName="instMag.i" serviceType="Poll" />
							<ExtRef intAddr="MMXU_Vol4" desc="sample_v4" iedName="TR2" ldInst="LD1" prefix="B2" lnClass="TVTR" lnInst="4" doName="VolSv" daName="instMag.i" serviceType="Poll" />
						</Inputs>
					</LN>
				</LDevice>
			</Server>
		</AccessPoint>
	</IED>
	<DataTypeTemplates>
		<LNodeType lnClass="PDIF" desc="Pdif1" id="PDIF$oscd$_ab16057220381a95">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="Op" type="Op$oscd$_8598a343000a4c8c" transient="true" />
			<DO name="LoSet" type="LoSet$oscd$_70f83e1c005744f2" />
			<DO name="HiSet" type="HiSet$oscd$_70f83e1c005744f2" />
			<DO name="MinOpTmms" type="MinOpTmms$oscd$_f1c0d370e7430e0c" />
			<DO name="MaxOpTmms" type="MaxOpTmms$oscd$_f1c0d370e7430e0c" />
			<DO name="RsDlTmms" type="RsDlTmms$oscd$_f1c0d370e7430e0c" />
			<DO name="RstMod" type="RstMod$oscd$_f291a9c0baf147b7" />
			<DO name="TmACrv" type="TmACrv$oscd$_4af833088bcc2499" />
		</LNodeType>
		<LNodeType lnClass="PTOC" desc="Ptoc1" id="PTOC$oscd$_72a47fe03fb65b2f">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="Op" type="Op$oscd$_8598a343000a4c8c" transient="true" />
			<DO name="Str" type="Str$oscd$_ba49448ecb46113a" />
			<DO name="RsDlTmms" type="RsDlTmms$oscd$_f1c0d370e7430e0c" />
			<DO name="OpDlTmms" type="OpDlTmms$oscd$_f1c0d370e7430e0c" />
			<DO name="MaxOpTmms" type="MaxOpTmms$oscd$_f1c0d370e7430e0c" />
			<DO name="MinOpTmms" type="MinOpTmms$oscd$_f1c0d370e7430e0c" />
			<DO name="StrVal" type="StrVal$oscd$_70f83e1c005744f2" />
			<DO name="TmACrv" type="TmACrv$oscd$_20fcfe62b7de20f2" />
			<DO name="TmMult" type="TmMult$oscd$_70f83e1c005744f2" />
			<DO name="TypRsCrv" type="TypRsCrv$oscd$_f291a9c0baf147b7" />
		</LNodeType>
		<LNodeType lnClass="PIOC" desc="Pioc1" id="PIOC$oscd$_c8c42d738d01930d">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="Op" type="Op$oscd$_8598a343000a4c8c" transient="true" />
			<DO name="StrVal" type="StrVal$oscd$_70f83e1c005744f2" />
		</LNodeType>
		<LNodeType lnClass="PTRC" desc="Ptrc1" id="PTRC$oscd$_a60344357bbc2c12">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="Op" type="Op$oscd$_8598a343000a4c8c" transient="true" />
			<DO name="Tr" type="Tr$oscd$_8598a343000a4c8c" />
		</LNodeType>
		<LNodeType lnClass="TVTR" desc="Tvtr1" id="TVTR$oscd$_e990c540f6cd3155">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="HzRtg" type="HzRtg$oscd$_24127d334358ed7c" />
			<DO name="VRtg" type="VRtg$oscd$_24127d334358ed7c" />
			<DO name="VolSv" type="VolSv$oscd$_0b87a0d7ebff5471" />
		</LNodeType>
		<LNodeType lnClass="TCTR" desc="Tctr1" id="TCTR$oscd$_5b652afdf7c13340">
			<DO name="ARtg" type="ARtg$oscd$_24127d334358ed7c" />
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="HzRtg" type="HzRtg$oscd$_70f83e1c005744f2" />
			<DO name="AmpSv" type="AmpSv$oscd$_0b87a0d7ebff5471" />
		</LNodeType>
		<LNodeType lnClass="RREC" desc="Rrec1" id="RREC$oscd$_804653c11bcce278">
			<DO name="AutoRecSt" type="AutoRecSt$oscd$_7dd4272b158cac6a" />
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="OpCls" type="OpCls$oscd$_8598a343000a4c8c" transient="true" />
		</LNodeType>
		<LNodeType lnClass="RDRE" desc="Rdre1" id="RDRE$oscd$_591008265cdcb958">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="FltNum" type="FltNum$oscd$_bbaa9369107884bc" />
			<DO name="RcdMade" type="RcdMade$oscd$_d915d66d9e42a575" transient="true" />
		</LNodeType>
		<LNodeType lnClass="RBDR" desc="Rbdr1" id="RBDR$oscd$_449644f40f4781fb">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="ChTrg" type="ChTrg$oscd$_d915d66d9e42a575" />
		</LNodeType>
		<LNodeType lnClass="RADR" desc="Radr1" id="RADR$oscd$_1b5ec9d7c1c56129">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="ChTrg" type="ChTrg$oscd$_d915d66d9e42a575" />
		</LNodeType>
		<LNodeType lnClass="MMXU" desc="Mmxu1" id="MMXU$oscd$_3abca1c29963576c">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="PhV" type="PhV$oscd$_6482021654d47858" />
			<DO name="A" type="A$oscd$_6482021654d47858" />
			<DO name="AvPhVPhs" type="AvPhVPhs$oscd$_bf0b950e5c38251f" />
			<DO name="AvAPhs" type="AvAPhs$oscd$_bf0b950e5c38251f" />
		</LNodeType>
		<LNodeType lnClass="CSWI" desc="Cswi1" id="CSWI$oscd$_a89dcc3919082155">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="Pos" type="Pos$oscd$_e3a102cb0a2b4b59" />
			<DO name="OpOpn" type="OpOpn$oscd$_8598a343000a4c8c" transient="true" />
			<DO name="OpCls" type="OpCls$oscd$_8598a343000a4c8c" transient="true" />
		</LNodeType>
		<LNodeType lnClass="CILO" desc="Cilo1" id="CILO$oscd$_aa7ec79ef27309b1">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="EnaCls" type="EnaCls$oscd$_d915d66d9e42a575" />
			<DO name="EnaOpn" type="EnaOpn$oscd$_d915d66d9e42a575" />
		</LNodeType>
		<LNodeType lnClass="XCBR" desc="Xcbr1" id="XCBR$oscd$_420e2f9e111e8bf2">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="BlkCls" type="BlkCls$oscd$_637b3bee56ce66e4" />
			<DO name="BlkOpn" type="BlkOpn$oscd$_637b3bee56ce66e4" />
			<DO name="Loc" type="Loc$oscd$_d915d66d9e42a575" />
			<DO name="OpCnt" type="OpCnt$oscd$_bbaa9369107884bc" />
			<DO name="Pos" type="Pos$oscd$_a672a31af87c3923" />
		</LNodeType>
		<LNodeType lnClass="XSWI" desc="Xswi1" id="XSWI$oscd$_430901aa6c264cc1">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="BlkCls" type="BlkCls$oscd$_637b3bee56ce66e4" />
			<DO name="BlkOpn" type="BlkOpn$oscd$_637b3bee56ce66e4" />
			<DO name="Loc" type="Loc$oscd$_d915d66d9e42a575" />
			<DO name="OpCnt" type="OpCnt$oscd$_bbaa9369107884bc" />
			<DO name="Pos" type="Pos$oscd$_a672a31af87c3923" />
			<DO name="SwTyp" type="SwTyp$oscd$_082f086f651ace0d" />
		</LNodeType>
		<LNodeType lnClass="LLN0" desc="LN0_1" id="LLN0$oscd$_86a723959df7af2e">
			<DO name="Beh" type="Beh$oscd$_c6ed035c8137b35a" />
			<DO name="Loc" type="Loc$oscd$_d915d66d9e42a575" />
		</LNodeType>
		<LNodeType lnClass="LPHD" desc="LPHD_1" id="LPHD$oscd$_8ca3cb769327538e">
			<DO name="PhyHealth" type="PhyHealth$oscd$_f60d28fe398302cb" />
			<DO name="PhyNam" type="PhyNam$oscd$_1b7440bbb0dea6d1" />
			<DO name="Proxy" type="Proxy$oscd$_d915d66d9e42a575" />
		</LNodeType>
		<DOType cdc="ASG" id="LoSet$oscd$_70f83e1c005744f2">
			<DA name="setMag" fc="SE" bType="Struct" type="setMag$oscd$_ed49c2f7a55ad05a" />
		</DOType>
		<DOType cdc="ASG" id="HiSet$oscd$_70f83e1c005744f2">
			<DA name="setMag" fc="SE" bType="Struct" type="setMag$oscd$_ed49c2f7a55ad05a" />
		</DOType>
		<DOType cdc="ENG" id="RstMod$oscd$_f291a9c0baf147b7">
			<DA name="setVal" fc="SE" bType="Enum" type="setVal$oscd$_98017d41cdb32bc1" />
		</DOType>
		<DOType cdc="CURVE" id="TmACrv$oscd$_4af833088bcc2499">
			<DA name="setCharact" fc="SE" bType="Enum" type="setCharact$oscd$_486b72b6f4047ed8" />
		</DOType>
		<DOType cdc="ING" id="RsDlTmms$oscd$_f1c0d370e7430e0c">
			<DA name="setVal" fc="SE" bType="INT32" />
		</DOType>
		<DOType cdc="ING" id="OpDlTmms$oscd$_f1c0d370e7430e0c">
			<DA name="setVal" fc="SE" bType="INT32" />
		</DOType>
		<DOType cdc="ING" id="MaxOpTmms$oscd$_f1c0d370e7430e0c">
			<DA name="setVal" fc="SE" bType="INT32" />
		</DOType>
		<DOType cdc="ING" id="MinOpTmms$oscd$_f1c0d370e7430e0c">
			<DA name="setVal" fc="SE" bType="INT32" />
		</DOType>
		<DOType cdc="CURVE" id="TmACrv$oscd$_20fcfe62b7de20f2">
			<DA name="setCharact" fc="SE" bType="Enum" type="setCharact$oscd$_95737bf2ad6f4abd" />
		</DOType>
		<DOType cdc="ASG" id="TmMult$oscd$_70f83e1c005744f2">
			<DA name="setMag" fc="SE" bType="Struct" type="setMag$oscd$_ed49c2f7a55ad05a" />
		</DOType>
		<DOType cdc="ENG" id="TypRsCrv$oscd$_f291a9c0baf147b7">
			<DA name="setVal" fc="SE" bType="Enum" type="setVal$oscd$_98017d41cdb32bc1" />
		</DOType>
		<DOType cdc="ASG" id="StrVal$oscd$_70f83e1c005744f2">
			<DA name="setMag" fc="SP" bType="Struct" type="setMag$oscd$_ed49c2f7a55ad05a" />
		</DOType>
		<DOType cdc="MV" id="AvPhVPhs$oscd$_bf0b950e5c38251f">
			<DA name="mag" fc="MX" dchg="true" dupd="true" bType="Struct" type="mag$oscd$_ed49c2f7a55ad05a" />
			<DA name="q" fc="MX" qchg="true" bType="Quality" />
			<DA name="t" fc="MX" bType="Timestamp" />
		</DOType>
		<DOType cdc="MV" id="AvAPhs$oscd$_bf0b950e5c38251f">
			<DA name="mag" fc="MX" dchg="true" dupd="true" bType="Struct" type="mag$oscd$_ed49c2f7a55ad05a" />
			<DA name="q" fc="MX" qchg="true" bType="Quality" />
			<DA name="t" fc="MX" bType="Timestamp" />
		</DOType>
		<DOType cdc="ACT" id="Tr$oscd$_8598a343000a4c8c">
			<DA name="general" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="SAV" id="VolSv$oscd$_0b87a0d7ebff5471">
			<DA name="instMag" fc="MX" bType="Struct" type="instMag$oscd$_5a5af9e249dc7f84" />
			<DA name="q" fc="MX" qchg="true" bType="Quality" />
		</DOType>
		<DOType cdc="SAV" id="AmpSv$oscd$_0b87a0d7ebff5471">
			<DA name="instMag" fc="MX" bType="Struct" type="instMag$oscd$_5a5af9e249dc7f84" />
			<DA name="q" fc="MX" qchg="true" bType="Quality" />
		</DOType>
		<DOType cdc="ENS" id="AutoRecSt$oscd$_7dd4272b158cac6a">
			<DA name="stVal" fc="ST" dchg="true" dupd="true" bType="Enum" type="stVal$oscd$_a3d45706712dddd7" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="INS" id="FltNum$oscd$_bbaa9369107884bc">
			<DA name="stVal" fc="ST" dchg="true" dupd="true" bType="INT32" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="SPS" id="RcdMade$oscd$_d915d66d9e42a575">
			<DA name="stVal" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="SPS" id="ChTrg$oscd$_d915d66d9e42a575">
			<DA name="stVal" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="CMV" id="phsA$oscd$_c63a8f4457479fb7">
			<DA name="cVal" fc="MX" dchg="true" dupd="true" bType="Struct" type="cVal$oscd$_21f679e08734a896" />
			<DA name="q" fc="MX" qchg="true" bType="Quality" />
			<DA name="t" fc="MX" bType="Timestamp" />
		</DOType>
		<DOType cdc="CMV" id="phsB$oscd$_c63a8f4457479fb7">
			<DA name="cVal" fc="MX" dchg="true" dupd="true" bType="Struct" type="cVal$oscd$_21f679e08734a896" />
			<DA name="q" fc="MX" qchg="true" bType="Quality" />
			<DA name="t" fc="MX" bType="Timestamp" />
		</DOType>
		<DOType cdc="CMV" id="phsC$oscd$_c63a8f4457479fb7">
			<DA name="cVal" fc="MX" dchg="true" dupd="true" bType="Struct" type="cVal$oscd$_21f679e08734a896" />
			<DA name="q" fc="MX" qchg="true" bType="Quality" />
			<DA name="t" fc="MX" bType="Timestamp" />
		</DOType>
		<DOType cdc="CMV" id="neut$oscd$_c63a8f4457479fb7">
			<DA name="cVal" fc="MX" dchg="true" dupd="true" bType="Struct" type="cVal$oscd$_21f679e08734a896" />
			<DA name="q" fc="MX" qchg="true" bType="Quality" />
			<DA name="t" fc="MX" bType="Timestamp" />
		</DOType>
		<DOType cdc="WYE" id="PhV$oscd$_6482021654d47858">
			<SDO name="neut" type="neut$oscd$_c63a8f4457479fb7" />
			<SDO name="phsC" type="phsC$oscd$_c63a8f4457479fb7" />
			<SDO name="phsB" type="phsB$oscd$_c63a8f4457479fb7" />
			<SDO name="phsA" type="phsA$oscd$_c63a8f4457479fb7" />
		</DOType>
		<DOType cdc="WYE" id="A$oscd$_6482021654d47858">
			<SDO name="neut" type="neut$oscd$_c63a8f4457479fb7" />
			<SDO name="phsC" type="phsC$oscd$_c63a8f4457479fb7" />
			<SDO name="phsB" type="phsB$oscd$_c63a8f4457479fb7" />
			<SDO name="phsA" type="phsA$oscd$_c63a8f4457479fb7" />
		</DOType>
		<DOType cdc="DPC" id="Pos$oscd$_e3a102cb0a2b4b59">
			<DA name="stVal" fc="ST" dchg="true" bType="Dbpos" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
			<DA name="ctlModel" fc="CF" dchg="true" bType="Enum" type="ctlModel$oscd$_f80264355419aeff" />
			<DA name="SBOw" fc="CO" bType="Struct" type="SBOw$oscd$_0d1ece84b67fe837" />
			<DA name="Oper" fc="CO" bType="Struct" type="Oper$oscd$_0d1ece84b67fe837" />
		</DOType>
		<DOType cdc="ACT" id="OpOpn$oscd$_8598a343000a4c8c">
			<DA name="general" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="ACT" id="OpCls$oscd$_8598a343000a4c8c">
			<DA name="general" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="SPS" id="EnaCls$oscd$_d915d66d9e42a575">
			<DA name="stVal" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="SPS" id="EnaOpn$oscd$_d915d66d9e42a575">
			<DA name="stVal" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="ACT" id="Op$oscd$_8598a343000a4c8c">
			<DA name="general" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="ACD" id="Str$oscd$_ba49448ecb46113a">
			<DA name="general" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="dirGeneral" fc="ST" dchg="true" bType="Enum" type="dirGeneral$oscd$_ba7b8abb9d154a3c" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="ASG" id="HzRtg$oscd$_24127d334358ed7c">
			<DA name="setMag" fc="SE" bType="Struct" type="setMag$oscd$_5a5af9e249dc7f84" />
		</DOType>
		<DOType cdc="ASG" id="VRtg$oscd$_24127d334358ed7c">
			<DA name="setMag" fc="SE" bType="Struct" type="setMag$oscd$_5a5af9e249dc7f84" />
		</DOType>
		<DOType cdc="ASG" id="ARtg$oscd$_24127d334358ed7c">
			<DA name="setMag" fc="SE" bType="Struct" type="setMag$oscd$_5a5af9e249dc7f84" />
		</DOType>
		<DOType cdc="ASG" id="HzRtg$oscd$_70f83e1c005744f2">
			<DA name="setMag" fc="SE" bType="Struct" type="setMag$oscd$_ed49c2f7a55ad05a" />
		</DOType>
		<DOType cdc="SPC" id="BlkCls$oscd$_637b3bee56ce66e4">
			<DA name="stVal" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
			<DA name="ctlModel" fc="CF" dchg="true" bType="Enum" type="ctlModel$oscd$_f80264355419aeff" />
		</DOType>
		<DOType cdc="SPC" id="BlkOpn$oscd$_637b3bee56ce66e4">
			<DA name="stVal" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
			<DA name="ctlModel" fc="CF" dchg="true" bType="Enum" type="ctlModel$oscd$_f80264355419aeff" />
		</DOType>
		<DOType cdc="SPS" id="Loc$oscd$_d915d66d9e42a575">
			<DA name="stVal" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="INS" id="OpCnt$oscd$_bbaa9369107884bc">
			<DA name="stVal" fc="ST" dchg="true" dupd="true" bType="INT32" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="DPC" id="Pos$oscd$_a672a31af87c3923">
			<DA name="stVal" fc="ST" dchg="true" bType="Dbpos" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
			<DA name="ctlModel" fc="CF" dchg="true" bType="Enum" type="ctlModel$oscd$_f80264355419aeff" />
		</DOType>
		<DOType cdc="ENS" id="SwTyp$oscd$_082f086f651ace0d">
			<DA name="stVal" fc="ST" dchg="true" dupd="true" bType="Enum" type="stVal$oscd$_e72af539034476a1" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="ENS" id="Beh$oscd$_c6ed035c8137b35a">
			<DA name="stVal" fc="ST" dchg="true" dupd="true" bType="Enum" type="stVal$oscd$_48ba16345b8e7f5b" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="ENS" id="PhyHealth$oscd$_f60d28fe398302cb">
			<DA name="stVal" fc="ST" dchg="true" dupd="true" bType="Enum" type="stVal$oscd$_0bcd07051dfc05a4" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DOType cdc="DPL" id="PhyNam$oscd$_1b7440bbb0dea6d1">
			<DA name="vendor" fc="DC" bType="VisString255" />
		</DOType>
		<DOType cdc="SPS" id="Proxy$oscd$_d915d66d9e42a575">
			<DA name="stVal" fc="ST" dchg="true" bType="BOOLEAN" />
			<DA name="q" fc="ST" qchg="true" bType="Quality" />
			<DA name="t" fc="ST" bType="Timestamp" />
		</DOType>
		<DAType id="instMag$oscd$_5a5af9e249dc7f84">
			<BDA name="i" bType="INT32" />
		</DAType>
		<DAType id="mag$oscd$_ed49c2f7a55ad05a">
			<BDA name="f" bType="FLOAT32" />
		</DAType>
		<DAType id="ang$oscd$_ed49c2f7a55ad05a">
			<BDA name="f" bType="FLOAT32" />
		</DAType>
		<DAType id="cVal$oscd$_21f679e08734a896">
			<BDA name="mag" bType="Struct" type="mag$oscd$_ed49c2f7a55ad05a" />
			<BDA name="ang" bType="Struct" type="ang$oscd$_ed49c2f7a55ad05a" />
		</DAType>
		<DAType id="origin$oscd$_cfc683368475eafc">
			<BDA name="orCat" bType="Enum" type="orCat$oscd$_711cd9acd4dad897" />
			<BDA name="orIdent" bType="Octet64" />
		</DAType>
		<DAType id="SBOw$oscd$_0d1ece84b67fe837">
			<BDA name="ctlVal" bType="BOOLEAN" />
			<BDA name="origin" bType="Struct" type="origin$oscd$_cfc683368475eafc" />
			<BDA name="ctlNum" bType="INT8U" />
			<BDA name="T" bType="Timestamp" />
			<BDA name="Test" bType="BOOLEAN" />
			<BDA name="Check" bType="Check" />
		</DAType>
		<DAType id="Oper$oscd$_0d1ece84b67fe837">
			<BDA name="ctlVal" bType="BOOLEAN" />
			<BDA name="origin" bType="Struct" type="origin$oscd$_cfc683368475eafc" />
			<BDA name="ctlNum" bType="INT8U" />
			<BDA name="T" bType="Timestamp" />
			<BDA name="Test" bType="BOOLEAN" />
			<BDA name="Check" bType="Check" />
		</DAType>
		<DAType id="setMag$oscd$_5a5af9e249dc7f84">
			<BDA name="i" bType="INT32" />
		</DAType>
		<DAType id="setMag$oscd$_ed49c2f7a55ad05a">
			<BDA name="f" bType="FLOAT32" />
		</DAType>
		<EnumType id="setCharact$oscd$_486b72b6f4047ed8">
			<EnumVal ord="15">IEC Definite Time</EnumVal>
		</EnumType>
		<EnumType id="setCharact$oscd$_95737bf2ad6f4abd">
			<EnumVal ord="11">IEC Inverse</EnumVal>
		</EnumType>
		<EnumType id="setVal$oscd$_98017d41cdb32bc1">
			<EnumVal ord="1">None</EnumVal>
		</EnumType>
		<EnumType id="stVal$oscd$_a3d45706712dddd7">
			<EnumVal ord="1">Ready</EnumVal>
			<EnumVal ord="2">InProgress</EnumVal>
			<EnumVal ord="3">Successful</EnumVal>
			<EnumVal ord="4">WaitingForTrip</EnumVal>
			<EnumVal ord="5">TripFromProtection</EnumVal>
			<EnumVal ord="6">FaultDisappeared</EnumVal>
			<EnumVal ord="7">WaitToComplete</EnumVal>
			<EnumVal ord="8">CBclosed</EnumVal>
			<EnumVal ord="9">CycleUnsuccessful</EnumVal>
			<EnumVal ord="10">Unsuccessful</EnumVal>
			<EnumVal ord="11">Aborted</EnumVal>
			<EnumVal ord="12">NotReady</EnumVal>
		</EnumType>
		<EnumType id="orCat$oscd$_711cd9acd4dad897">
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
		<EnumType id="dirGeneral$oscd$_ba7b8abb9d154a3c">
			<EnumVal ord="0">unknown</EnumVal>
			<EnumVal ord="1">forward</EnumVal>
			<EnumVal ord="2">backward</EnumVal>
			<EnumVal ord="3">both</EnumVal>
		</EnumType>
		<EnumType id="ctlModel$oscd$_f80264355419aeff">
			<EnumVal ord="0">status-only</EnumVal>
			<EnumVal ord="1">direct-with-normal-security</EnumVal>
			<EnumVal ord="2">sbo-with-normal-security</EnumVal>
			<EnumVal ord="3">direct-with-enhanced-security</EnumVal>
			<EnumVal ord="4">sbo-with-enhanced-security</EnumVal>
		</EnumType>
		<EnumType id="stVal$oscd$_e72af539034476a1">
			<EnumVal ord="1">Load Break</EnumVal>
			<EnumVal ord="2">Disconnector</EnumVal>
			<EnumVal ord="3">Earthing Switch</EnumVal>
			<EnumVal ord="4">High Speed Earthing Switch</EnumVal>
		</EnumType>
		<EnumType id="stVal$oscd$_48ba16345b8e7f5b">
			<EnumVal ord="1">on</EnumVal>
			<EnumVal ord="2">blocked</EnumVal>
			<EnumVal ord="3">test</EnumVal>
			<EnumVal ord="4">test/blocked</EnumVal>
			<EnumVal ord="5">off</EnumVal>
		</EnumType>
		<EnumType id="stVal$oscd$_0bcd07051dfc05a4">
			<EnumVal ord="1">Ok</EnumVal>
			<EnumVal ord="2">Warning</EnumVal>
			<EnumVal ord="3">Alarm</EnumVal>
		</EnumType>
	</DataTypeTemplates>
</SCL>