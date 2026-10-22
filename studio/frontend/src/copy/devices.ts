// Wording of the Devices place (change 0177): why a button is grey while running, and small plain facts.

export const DEVICES_COPY = {
  importLocked: 'Stop the run to import IEDs',
  sampleLocked: 'Stop the run to add a sample IED',
  roleLocked: 'Stop the run to change what ScadaBench does with this device',
  noAddress: 'not in the file',
};

/** The import error the engine words for experts, with a plain explanation and what to do. The engine's
 *  text stays as it is (shared with the Python engine); only the page shows the friendlier one. */
const FRIENDLY_IMPORT: Record<string, string> = {
  'Missing DataTypeTemplates': 'This file has no data type section (DataTypeTemplates), so ScadaBench cannot build the IED model. Export the SCD or CID again from your engineering tool with the data types included.',
};

export const friendlyImportError = (message: string) => FRIENDLY_IMPORT[message.trim()] ?? message;
