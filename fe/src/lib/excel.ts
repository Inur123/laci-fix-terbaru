import * as SheetJS from "xlsx";
import writeXlsxFile, { type SheetData } from "write-excel-file/browser";

type Style = {
  font?: { bold?: boolean; name?: string; color?: { rgb?: string } };
  fill?: { fgColor?: { rgb?: string } };
  alignment?: { horizontal?: string; vertical?: string };
  border?: { top?: { style?: string; color?: { rgb?: string } } };
};

type StyledCell = SheetJS.CellObject & { s?: Style };

function color(rgb?: string) {
  return rgb ? `#${rgb.slice(-6)}` : undefined;
}

async function writeFile(workbook: SheetJS.WorkBook, filename: string) {
  const worksheet = workbook.Sheets[workbook.SheetNames[0]];
  if (!worksheet) throw new Error("Lembar Excel tidak ditemukan");

  const rawRows = SheetJS.utils.sheet_to_json<(string | number | boolean | Date | null)[]>(worksheet, {
    header: 1,
    raw: true,
    defval: null,
  });
  const rows = rawRows.map((row, rowIndex) => row.map((value, columnIndex) => {
    if (rowIndex !== 0) return value;
    const address = SheetJS.utils.encode_cell({ r: rowIndex, c: columnIndex });
    const style = (worksheet[address] as StyledCell | undefined)?.s;
    if (!style) return value;
    return {
      value,
      fontWeight: style.font?.bold ? "bold" : undefined,
      fontFamily: style.font?.name,
      textColor: color(style.font?.color?.rgb),
      backgroundColor: color(style.fill?.fgColor?.rgb),
      align: style.alignment?.horizontal,
      alignVertical: style.alignment?.vertical,
      borderStyle: style.border?.top?.style,
      borderColor: color(style.border?.top?.color?.rgb),
    };
  }));
  const columns = (worksheet["!cols"] ?? []).map((column) => ({ width: column.wch ?? 14 }));
  await writeXlsxFile(rows as unknown as SheetData, { columns }).toFile(filename);
}

const XLSX = { utils: SheetJS.utils, writeFile };
export default XLSX;
