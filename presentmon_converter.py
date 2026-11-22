import csv
import os
import sys

def convert_presentmon(src_file, dst_file):
    print(f"Reading: {src_file}")
    
    with open(src_file, 'r', encoding='utf-8-sig') as f_in, open(dst_file, 'w', newline='', encoding='utf-8') as f_out:
        reader = csv.DictReader(f_in)
        writer = csv.writer(f_out)
        
        # Write the header expected by the Go tool
        writer.writerow(["frame", "frametime_ms"])
        
        frame_count = 0
        skipped_na = 0
        skipped_non_app = 0
        skipped_invalid = 0
        
        for row in reader:
            # 1. Filter for Application frames only (ignore repeated/dropped frames)
            frame_type = row.get("FrameType", "Application")
            if frame_type != "Application":
                skipped_non_app += 1
                continue
                
            # 2. Extract the frametime
            ms_str = row.get("MsBetweenPresents", "NA").strip()
            
            # 3. Handle PresentMon's NA values
            if ms_str == "NA" or ms_str == "":
                skipped_na += 1
                continue
                
            try:
                ms_val = float(ms_str)
                # 4. Ensure it's a valid positive number (Go tool rejects <= 0)
                if ms_val <= 0:
                    skipped_invalid += 1
                    continue
            except ValueError:
                skipped_invalid += 1
                continue
                
            # 5. Write the valid row
            frame_count += 1
            writer.writerow([frame_count, f"{ms_val:.4f}"])
            
    print(f"Done! Wrote {frame_count} frames to {dst_file}")
    print(f"Skipped: {skipped_na} NA values, {skipped_non_app} non-Application frames, {skipped_invalid} invalid/zero values.")
    
    if frame_count < 1000:
        print("\nWARNING: Less than 1000 frames recorded.")
        print("The Go tool will not be able to calculate a 0.1% low.")
    if frame_count < 100:
        print("\nWARNING: Less than 100 frames recorded.")
        print("The Go tool will not be able to calculate a 1% low.")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python presentmon_converter.py <input_pmcap.csv> [output.csv]")
        print("Example: python presentmon_converter.py data/src/pmcap-LimbusCompany.exe-261121-220342.csv data/dst/limbus_frametimes.csv")
        sys.exit(1)
        
    input_path = sys.argv[1]
    
    # Generate output path if not provided
    if len(sys.argv) > 2:
        output_path = sys.argv[2]
    else:
        output_path = input_path.replace(".csv", "_frametimes.csv")
        
    if not os.path.exists(input_path):
        print(f"Error: File not found: {input_path}")
        sys.exit(1)
        
    convert_presentmon(input_path, output_path)