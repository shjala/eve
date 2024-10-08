"""
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2024 Zededa, Inc.

This script reads the log file generated by the statistics collector and visualizes the memory
pressure over time. The script uses Plotly to create interactive plots that can be viewed in a
web browser.
"""

import sys
import os

import pandas as pd
import plotly.graph_objects as go

EXPECTED_HEADER = 'date time someAvg10 someAvg60 someAvg300 someTotal ' \
                  'fullAvg10 fullAvg60 fullAvg300 fullTotal'


def visualize_memory_pressure(log_file):
    """
    Visualizes the memory pressure over time using an interactive plot.
    :param log_file: Path to the log file generated by the statistics collector.
    :return: None
    """
    # Read the log file into a DataFrame
    dataframe = pd.read_csv(log_file, sep=r'\s+')

    # Combine 'date' and 'time' columns into a single 'Timestamp' column
    dataframe['Timestamp'] = pd.to_datetime(dataframe['date'] + ' ' + dataframe['time'],
                                            format='%Y-%m-%d %H:%M:%S')

    # Drop the now redundant 'date' and 'time' columns
    dataframe.drop(columns=['date', 'time'], inplace=True)

    # Create interactive plots using Plotly
    fig = go.Figure()

    # Adding traces for 'some' values
    fig.add_trace(go.Scatter(x=dataframe['Timestamp'], y=dataframe['someAvg10'], mode='lines',
                             name='someAvg10', yaxis="y1"))
    fig.add_trace(go.Scatter(x=dataframe['Timestamp'], y=dataframe['someAvg60'], mode='lines',
                             name='someAvg60', yaxis="y1"))
    fig.add_trace(go.Scatter(x=dataframe['Timestamp'], y=dataframe['someAvg300'], mode='lines',
                             name='someAvg300', yaxis="y1"))

    # Adding traces for 'full' values
    fig.add_trace(go.Scatter(x=dataframe['Timestamp'], y=dataframe['fullAvg10'], mode='lines',
                             name='fullAvg10', yaxis="y1"))
    fig.add_trace(go.Scatter(x=dataframe['Timestamp'], y=dataframe['fullAvg60'], mode='lines',
                             name='fullAvg60', yaxis="y1"))
    fig.add_trace(go.Scatter(x=dataframe['Timestamp'], y=dataframe['fullAvg300'], mode='lines',
                             name='fullAvg300', yaxis="y1"))

    # Adding cumulative area plots for total values
    fig.add_trace(go.Scatter(x=dataframe['Timestamp'], y=dataframe['someTotal'], mode='lines',
                             name='someTotal', line={"width": 0.5, "color": 'rgb(131, 90, 241)'},
                             stackgroup='one', yaxis="y2"))  # for area plot
    fig.add_trace(go.Scatter(x=dataframe['Timestamp'], y=dataframe['fullTotal'], mode='lines',
                             name='fullTotal', line={"width": 0.5, "color": 'rgb(255, 50, 50)'},
                             stackgroup='two', yaxis="y2"))  # for area plot

    # Update layout for better readability
    fig.update_layout(
        title="Memory Pressure Over Time",
        xaxis_title="Timestamp",
        yaxis_title="Values",
        yaxis={"range": [0, 100], "title": "Pressure Averages"},
        yaxis2={"title": "Total Values", "overlaying": "y", "side": "right"},
        legend_title="Metrics",
        hovermode="x unified"
    )

    # Save the plot as an HTML file
    fig.write_html("memory_pressure.html")

    # Show the interactive plot
    fig.show()


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python visualize.py <log_file>")
        sys.exit(1)

    # Check if the log file exists
    if not os.path.exists(sys.argv[1]):
        print(f"Error: Log file '{sys.argv[1]}' not found!")
        sys.exit(1)

    # Check the header of the log file
    with open(sys.argv[1], encoding='utf-8') as f:
        header = f.readline().strip()
        if header != EXPECTED_HEADER:
            print(f"Error: Invalid log file '{sys.argv[1]}'!")
            sys.exit(1)

    LOG_FILE_ARG = sys.argv[1]
    visualize_memory_pressure(LOG_FILE_ARG)
